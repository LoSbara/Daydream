package agents

import (
	"context"
	"fmt"
	"log"
	"strings"

	"daydream/internal/db"
	"daydream/internal/embedding"
	"daydream/internal/llm"
	"daydream/internal/models"
	"daydream/internal/rag"
)

// ContentGenerator genera nuovi documenti nella knowledge_base in modo asincrono.
// Viene invocato post-turno quando il GM emette "content_gen" nel JSON.
// Prima di generare, interroga la KB per il contesto correlato, garantendo coerenza
// tra ciò che è già scritto e il nuovo contenuto.
type ContentGenerator struct {
	db        *db.Client
	llm       llm.Provider
	embed     embedding.Provider
	retriever *rag.Retriever
}

func NewContentGenerator(database *db.Client, provider llm.Provider, embedder embedding.Provider, retriever *rag.Retriever) *ContentGenerator {
	return &ContentGenerator{
		db:        database,
		llm:       provider,
		embed:     embedder,
		retriever: retriever,
	}
}

// GenerateAsync avvia la generazione in background per ogni richiesta. Non blocca il turno.
func (cg *ContentGenerator) GenerateAsync(ctx context.Context, requests []models.ContentGenRequest, charID string) {
	go func() {
		for _, req := range requests {
			if req.Subject == "" || req.Type == "" {
				continue
			}
			if err := cg.generate(context.Background(), req, charID); err != nil {
				log.Printf("[content-gen] errore generazione %s '%s': %v", req.Type, req.Subject, err)
			}
		}
	}()
}

func (cg *ContentGenerator) generate(ctx context.Context, req models.ContentGenRequest, charID string) error {
	entryID := buildEntryID(req.Type, req.Subject)

	// 1. Idempotenza: salta se il documento esiste già
	if exists, _ := cg.entryExists(entryID); exists {
		log.Printf("[content-gen] '%s' già presente in KB, skip", entryID)
		return nil
	}

	// 2. Leggi la KB per contesto correlato — garantisce coerenza con ciò che esiste
	relatedDocs := ""
	if cg.retriever != nil {
		entries := cg.retriever.Retrieve(ctx, req.Subject+" "+req.Context, 3)
		if len(entries) > 0 {
			var sb strings.Builder
			for _, e := range entries {
				sb.WriteString(fmt.Sprintf("### %s\n%s\n\n", e.Title, e.Content))
			}
			relatedDocs = sb.String()
		}
	}

	// 3. Genera il documento via LLM
	content, title, err := cg.callLLM(ctx, req, relatedDocs)
	if err != nil {
		return fmt.Errorf("generazione LLM: %w", err)
	}

	// 4. Calcola embedding
	vecs, err := cg.embed.Embed(ctx, []string{title+"\n"+content})
	if err != nil {
		return fmt.Errorf("embedding: %w", err)
	}
	vec := vecs[0]
	vecAny := make([]any, len(vec))
	for i, v := range vec {
		vecAny[i] = v
	}

	// 5. Inserisci nella KB
	sql := fmt.Sprintf(`CREATE knowledge_base:%s CONTENT {
		"title":     %s,
		"content":   %s,
		"category":  %s,
		"embedding": %s
	}`, entryID, jsonStr(title), jsonStr(content), jsonStr(req.Type), mustMarshal(vecAny))

	if err := cg.db.Exec(sql, nil); err != nil {
		return fmt.Errorf("insert KB: %w", err)
	}

	log.Printf("[content-gen] generato documento KB '%s' (tipo: %s)", entryID, req.Type)
	return nil
}

func (cg *ContentGenerator) callLLM(ctx context.Context, req models.ContentGenRequest, relatedDocs string) (content, title string, err error) {
	typeDescriptions := map[string]string{
		"npc":           "un personaggio non giocante (NPC)",
		"zone":          "una zona o area geografica del mondo",
		"dungeon":       "un dungeon o struttura esplorabile",
		"lore":          "un evento, storia o concetto del mondo",
		"quest_context": "il contesto narrativo di una quest",
	}
	typeDesc := typeDescriptions[req.Type]
	if typeDesc == "" {
		typeDesc = "un elemento del mondo"
	}

	systemPrompt := `Sei un world-builder esperto per un VRMMO fantasy dark chiamato Daydream.
Il tuo compito è creare documenti di lore dettagliati e coerenti per la Knowledge Base del gioco.
Questi documenti verranno usati da un GM AI per narrare il gioco in modo consistente.

REGOLE ASSOLUTE:
1. Sii coerente con i documenti esistenti forniti come contesto.
2. Non contraddire mai informazioni già stabilite nella KB.
3. Scrivi in italiano, stile narrativo-descrittivo, tono dark fantasy.
4. Lunghezza ideale: 150-300 parole. Non di più, non di meno.
5. Includi dettagli concreti: nomi, luoghi, relazioni, motivazioni, segreti.
6. Per NPC: include aspetto fisico, personalità, ruolo nel mondo, relazioni con altri NPC noti.
7. Per zone/dungeon: include atmosfera, pericoli tipici, storia del luogo, cosa ci si può trovare.
8. Rispondi SOLO con il documento, senza intestazioni meta o spiegazioni.`

	contextSection := ""
	if relatedDocs != "" {
		contextSection = "\n\n## DOCUMENTI CORRELATI ESISTENTI (rispetta la coerenza con questi)\n" + relatedDocs
	}

	userPrompt := fmt.Sprintf(`Crea un documento di lore per %s chiamato/a "%s".

Descrizione di partenza fornita dal GM: %s
%s

Scrivi un documento completo e dettagliato adatto alla Knowledge Base.`, typeDesc, req.Subject, req.Context, contextSection)

	msgs := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	raw, err := cg.llm.Complete(ctx, msgs, llm.CompleteOpts{MaxTokens: 600, Temperature: 0.75})
	if err != nil {
		return "", "", err
	}

	content = strings.TrimSpace(raw)
	title = buildTitle(req.Type, req.Subject)
	return content, title, nil
}

func (cg *ContentGenerator) entryExists(id string) (bool, error) {
	results, err := cg.db.Query(
		"SELECT id FROM knowledge_base WHERE id = type::record('knowledge_base', $id)",
		map[string]any{"id": id},
	)
	if err != nil {
		return false, err
	}
	var rows []map[string]any
	if err := results[0].All(&rows); err != nil {
		return false, nil
	}
	return len(rows) > 0, nil
}

// buildEntryID genera un ID stabile e unico per la KB a partire da tipo e soggetto.
func buildEntryID(entryType, subject string) string {
	slug := strings.ToLower(subject)
	slug = strings.ReplaceAll(slug, " ", "_")
	// Rimuovi caratteri non alfanumerici eccetto underscore
	var b strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	return entryType + "_" + b.String() + "_gen"
}

func buildTitle(entryType, subject string) string {
	prefixes := map[string]string{
		"npc":           subject + " — Personaggio",
		"zone":          subject + " — Zona",
		"dungeon":       subject + " — Dungeon",
		"lore":          subject,
		"quest_context": subject + " — Contesto Quest",
	}
	if t, ok := prefixes[entryType]; ok {
		return t
	}
	return subject
}

func mustMarshal(v any) string {
	// Serializza un []any come array JSON
	items, ok := v.([]any)
	if !ok {
		return "[]"
	}
	var sb strings.Builder
	sb.WriteRune('[')
	for i, item := range items {
		if i > 0 {
			sb.WriteRune(',')
		}
		switch val := item.(type) {
		case float32:
			sb.WriteString(fmt.Sprintf("%g", val))
		case float64:
			sb.WriteString(fmt.Sprintf("%g", val))
		default:
			sb.WriteString(fmt.Sprintf("%v", val))
		}
	}
	sb.WriteRune(']')
	return sb.String()
}
