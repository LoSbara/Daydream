package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"daydream/internal/db"
	"daydream/internal/embedding"
)

// SeedEntry è un documento da inserire nella knowledge_base.
type SeedEntry struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Category string `json:"category"`
}

// Seeder carica i documenti lore nella knowledge_base con embedding.
type Seeder struct {
	db    db.DBClient
	embed embedding.Provider
}

func NewSeeder(database db.DBClient, embedder embedding.Provider) *Seeder {
	return &Seeder{db: database, embed: embedder}
}

// Seed legge knowledge_base.json, calcola gli embedding e inserisce i documenti.
// È idempotente: salta i documenti già presenti.
func (s *Seeder) Seed(ctx context.Context) error {
	entries, err := loadSeedFile()
	if err != nil {
		return fmt.Errorf("load knowledge_base.json: %w", err)
	}

	inserted := 0
	skipped := 0

	// Raggruppa in batch per embedding
	const batchSize = 20
	for i := 0; i < len(entries); i += batchSize {
		end := i + batchSize
		if end > len(entries) {
			end = len(entries)
		}
		batch := entries[i:end]

		// Filtra già esistenti
		toEmbed := make([]SeedEntry, 0, len(batch))
		for _, e := range batch {
			exists, err := s.exists(e.ID)
			if err != nil {
				return fmt.Errorf("check exists %s: %w", e.ID, err)
			}
			if exists {
				skipped++
			} else {
				toEmbed = append(toEmbed, e)
			}
		}

		if len(toEmbed) == 0 {
			continue
		}

		// Calcola embedding in batch
		texts := make([]string, len(toEmbed))
		for j, e := range toEmbed {
			texts[j] = e.Title + "\n" + e.Content
		}

		vecs, err := s.embed.Embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("embed batch: %w", err)
		}

		// Inserisci nel DB
		for j, e := range toEmbed {
			vec := vecs[j]
			vecAny := make([]any, len(vec))
			for k, v := range vec {
				vecAny[k] = v
			}

			sql := fmt.Sprintf(`CREATE knowledge_base:%s CONTENT {
				"title":     %s,
				"content":   %s,
				"category":  %s,
				"embedding": %s
			}`,
				e.ID,
				jsonStr(e.Title),
				jsonStr(e.Content),
				jsonStr(e.Category),
				mustMarshal(vecAny),
			)
			if err := s.db.Exec(sql, nil); err != nil {
				return fmt.Errorf("insert kb entry %s: %w", e.ID, err)
			}
			inserted++
		}
	}

	log.Printf("knowledge_base seeding: %d inseriti, %d già presenti", inserted, skipped)
	return nil
}

func (s *Seeder) exists(id string) (bool, error) {
	qr, err := s.db.QueryOne(
		"SELECT id FROM knowledge_base WHERE id = type::record('knowledge_base', $id)",
		map[string]any{"id": id},
	)
	if err != nil {
		return false, err
	}
	var rows []map[string]any
	if err := qr.All(&rows); err != nil {
		return false, nil
	}
	return len(rows) > 0, nil
}

func loadSeedFile() ([]SeedEntry, error) {
	path := kbSeedPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []SeedEntry
	return entries, json.Unmarshal(data, &entries)
}

func kbSeedPath() string {
	if env := os.Getenv("KB_SEED_PATH"); env != "" {
		return env
	}
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "configs", "seeds", "knowledge_base.json")
}

func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func mustMarshal(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
