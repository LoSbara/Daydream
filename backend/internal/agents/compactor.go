package agents

import (
	"context"
	"fmt"
	"log"
	"daydream/internal/db"
	"daydream/internal/llm"
	"strings"
)

const compactThreshold = 3200 // ~800 token

// Compactor comprime context_memo quando supera la soglia, in modo asincrono.
type Compactor struct {
	db  db.DBClient
	llm llm.Provider
}

func NewCompactor(database db.DBClient, provider llm.Provider) *Compactor {
	return &Compactor{db: database, llm: provider}
}

// CompactIfNeeded avvia la compaction in background se il memo supera la soglia.
// Non blocca il turno corrente.
func (c *Compactor) CompactIfNeeded(ctx context.Context, sessionID, memo string) {
	if len(memo) <= compactThreshold {
		return
	}
	log.Printf("[compactor] memo lungo %d char, avvio compaction per session %s", len(memo), sessionID)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[compactor] panic recuperato per session %s: %v", sessionID, r)
			}
		}()
		c.compact(context.Background(), sessionID, memo)
	}()
}

func (c *Compactor) compact(ctx context.Context, sessionID, memo string) {
	msgs := []llm.Message{
		{
			Role: "system",
			Content: `Sei un assistente specializzato nella compressione di note di sessione per un VRMMO.
Il tuo compito: comprimere il MEMO mantenendo TUTTI i fatti meccanici rilevanti:
- Accordi presi con NPC, transazioni economiche, debiti/crediti
- Quest accettate, completate o fallite e loro stato attuale
- NPC incontrati con nome e ruolo narrativo rilevante
- Flags di gioco importanti (boss sconfitti, zone sbloccate, eventi speciali)
- Stato delle relazioni del personaggio con fazioni e NPC

Elimina: ridondanze, descrizioni decorative, eventi già completamente risolti, ripetizioni.
Output: SOLO il testo compresso. Massimo 400 token. Nessuna spiegazione aggiuntiva.`,
		},
		{
			Role:    "user",
			Content: "MEMO DA COMPRIMERE:\n" + memo,
		},
	}

	compressed, err := c.llm.Complete(ctx, msgs, llm.CompleteOpts{MaxTokens: 500, Temperature: 0.3})
	if err != nil {
		log.Printf("[compactor] errore LLM per session %s: %v", sessionID, err)
		return
	}

	compressed = strings.TrimSpace(compressed)
	if compressed == "" {
		log.Printf("[compactor] risposta vuota per session %s", sessionID)
		return
	}

	sql := fmt.Sprintf("UPDATE %s SET context_memo = %s;", sessionID, jsonStr(compressed))
	if err := c.db.Exec(sql, nil); err != nil {
		log.Printf("[compactor] errore aggiornamento DB session %s: %v", sessionID, err)
		return
	}

	log.Printf("[compactor] memo compresso: %d → %d char (session %s)", len(memo), len(compressed), sessionID)
}
