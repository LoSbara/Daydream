package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Seed carica i dati seed (design pipeline) nel database.
// È idempotente: salta i record già esistenti.
func (c *Client) Seed() error {
	if err := c.seedSkills(); err != nil {
		return fmt.Errorf("seed skills: %w", err)
	}
	if err := c.seedCatalog("classes.json", "class_catalog"); err != nil {
		return fmt.Errorf("seed classes: %w", err)
	}
	if err := c.seedCatalog("monsters.json", "monster_catalog"); err != nil {
		return fmt.Errorf("seed monsters: %w", err)
	}
	return nil
}

// seedSkills carica il catalogo skill da configs/seeds/skills.json.
func (c *Client) seedSkills() error {
	path := seedsPath("skills.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read skills.json: %w", err)
	}

	var skills []map[string]any
	if err := json.Unmarshal(data, &skills); err != nil {
		return fmt.Errorf("parse skills.json: %w", err)
	}

	inserted := 0
	skipped := 0

	for _, skill := range skills {
		id, ok := skill["id"].(string)
		if !ok || id == "" {
			continue
		}

		// Controlla se esiste già
		qr, err := c.QueryOne("SELECT id FROM skill_catalog WHERE id = type::record('skill_catalog', $sid)",
			map[string]any{"sid": id})
		if err != nil {
			return fmt.Errorf("check skill %s: %w", id, err)
		}
		var existing []map[string]any
		if qr.All(&existing) == nil && len(existing) > 0 {
			skipped++
			continue
		}

		// Inserisci
		skillBytes, err := json.Marshal(skill)
		if err != nil {
			return err
		}
		sql := fmt.Sprintf("INSERT INTO skill_catalog %s;", string(skillBytes))
		if err := c.Exec(sql, nil); err != nil {
			return fmt.Errorf("insert skill %s: %w", id, err)
		}
		inserted++
	}

	_ = skipped // log opzionale
	_ = inserted
	return nil
}

// seedCatalog carica un file JSON generico in una tabella SCHEMALESS.
// Ogni record deve avere un campo "id" string. È idempotente: salta i record già presenti.
func (c *Client) seedCatalog(filename, table string) error {
	path := seedsPath(filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", filename, err)
	}

	var records []map[string]any
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("parse %s: %w", filename, err)
	}

	inserted := 0
	skipped := 0

	for _, rec := range records {
		id, ok := rec["id"].(string)
		if !ok || id == "" {
			continue
		}

		qr, err := c.QueryOne(
			fmt.Sprintf("SELECT id FROM %s WHERE id = type::record('%s', $sid)", table, table),
			map[string]any{"sid": id},
		)
		if err != nil {
			return fmt.Errorf("check %s/%s: %w", table, id, err)
		}
		var existing []map[string]any
		if qr.All(&existing) == nil && len(existing) > 0 {
			skipped++
			continue
		}

		recBytes, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		sql := fmt.Sprintf("INSERT INTO %s %s;", table, string(recBytes))
		if err := c.Exec(sql, nil); err != nil {
			return fmt.Errorf("insert %s/%s: %w", table, id, err)
		}
		inserted++
	}

	_ = skipped
	_ = inserted
	return nil
}

// seedsPath restituisce il percorso assoluto dei file di seed.
func seedsPath(filename string) string {
	// Se SEEDS_PATH è impostato, usalo
	if env := os.Getenv("SEEDS_PATH"); env != "" {
		return filepath.Join(env, filename)
	}
	// Default: relativo a questo file → ../../configs/seeds/
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "configs", "seeds", filename)
}
