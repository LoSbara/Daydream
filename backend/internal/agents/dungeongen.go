package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"daydream/internal/llm"
	"daydream/internal/models"
)

// DungeonGenerator arricchisce le stanze di un dungeon con contenuto AI-generato.
// La struttura meccanica (exits, has_enemy, tier) è deterministica e server-side;
// l'AI produce solo nomi evocativi e descrizioni atmosferiche.
type DungeonGenerator struct {
	llm llm.Provider
}

func NewDungeonGenerator(provider llm.Provider) *DungeonGenerator {
	return &DungeonGenerator{llm: provider}
}

type dungeonRoomContent struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type dungeonRoomsWrapper struct {
	Rooms []dungeonRoomContent `json:"rooms"`
}

// EnrichRooms sostituisce nomi e descrizioni statici delle stanze con contenuto LLM.
// Modifica la mappa in-place. In caso di errore o timeout le stanze mantengono
// il contenuto statico generato da GenerateDungeon.
// Blocking con timeout di 12s — da chiamare prima di rispondere al client.
func (g *DungeonGenerator) EnrichRooms(ctx context.Context, dungeonName string, difficulty int, rooms map[string]models.DungeonRoom) {
	enrichCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	if err := g.enrich(enrichCtx, dungeonName, difficulty, rooms); err != nil {
		log.Printf("[dungeon-gen] enrichment fallito per %q (fallback ai nomi statici): %v", dungeonName, err)
	}
}

func (g *DungeonGenerator) enrich(ctx context.Context, dungeonName string, difficulty int, rooms map[string]models.DungeonRoom) error {
	// Costruisci lista stanze ordinata con tipo meccanico per il prompt
	type roomMeta struct {
		id       string
		roomType string
	}
	var ordered []roomMeta
	for id, r := range rooms {
		rt := "stanza normale"
		switch {
		case r.IsEntrance:
			rt = "ingresso (sicuro, nessun nemico)"
		case r.IsBoss:
			rt = "stanza boss finale (massimo pericolo)"
		case r.EnemyTier == "elite":
			rt = "stanza con nemico elite"
		case r.HasEnemy:
			rt = "stanza con nemico normale"
		}
		ordered = append(ordered, roomMeta{id: id, roomType: rt})
	}

	// Build lista stanze per il prompt
	var roomLines strings.Builder
	for _, rm := range ordered {
		roomLines.WriteString(fmt.Sprintf("- %s [%s]\n", rm.id, rm.roomType))
	}

	msgs := []llm.Message{
		{
			Role: "system",
			Content: `Sei un dungeon designer per un VRMMO dark fantasy chiamato Daydream.
Devi generare nomi evocativi e descrizioni atmosferiche per le stanze di un dungeon.

REGOLE:
1. Nomi di 2-4 parole, evocativi e tematicamente coerenti col dungeon
2. Descrizioni di 25-40 parole in seconda persona ("Senti...", "L'aria...", "Qualcosa si muove...")
3. Tono dark fantasy hardcore, italiano
4. NON rivelare i nemici nella descrizione — solo atmosfera e tensione
5. Stanza boss: trasmetti senso di minaccia imminente, silenzio innaturale
6. Ingresso: primo impatto, moderatamente sicuro ma inquietante
7. Coerenza tematica: tutte le stanze devono condividere l'atmosfera del dungeon

Output SOLO come JSON oggetto con chiave "rooms", nessun testo extra.`,
		},
		{
			Role: "user",
			Content: fmt.Sprintf(
				"DUNGEON: %s (Difficoltà %d)\n\nSTANZE:\n%s\nRispondi con:\n{\"rooms\":[{\"id\":\"room_00\",\"name\":\"...\",\"description\":\"...\"},...]}",
				dungeonName, difficulty, roomLines.String(),
			),
		},
	}

	raw, err := g.llm.Complete(ctx, msgs, llm.CompleteOpts{
		MaxTokens:   1200,
		Temperature: 0.8,
		JSONMode:    true,
	})
	if err != nil {
		return fmt.Errorf("LLM: %w", err)
	}

	raw = strings.TrimSpace(raw)

	// Tenta il parse come wrapper oggetto {"rooms":[...]}
	var wrapper dungeonRoomsWrapper
	if err := json.Unmarshal([]byte(raw), &wrapper); err != nil {
		// Fallback: cerca array diretto [...] all'interno del raw
		start := strings.Index(raw, "[")
		end := strings.LastIndex(raw, "]")
		if start < 0 || end < 0 {
			return fmt.Errorf("risposta LLM non contiene JSON array: %.200s", raw)
		}
		arrayRaw := raw[start : end+1]
		if err := json.Unmarshal([]byte(arrayRaw), &wrapper.Rooms); err != nil {
			return fmt.Errorf("parse JSON array: %w (raw: %.200s)", err, arrayRaw)
		}
	}

	// Applica i contenuti generati alle stanze (merge in-place)
	updated := 0
	for _, c := range wrapper.Rooms {
		if c.ID == "" {
			continue
		}
		r, ok := rooms[c.ID]
		if !ok {
			continue
		}
		if c.Name != "" {
			r.Name = c.Name
		}
		if c.Description != "" {
			r.Description = c.Description
		}
		rooms[c.ID] = r
		updated++
	}

	log.Printf("[dungeon-gen] arricchite %d/%d stanze per %q", updated, len(rooms), dungeonName)
	return nil
}
