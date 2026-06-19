package game

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"daydream/internal/db"
)

type ledgerEntry struct {
	CharacterID  string    `json:"character_id"`
	TurnID       int       `json:"turn_id"`
	Type         string    `json:"type"`         // gain | loss | penalty | loot
	Amount       int       `json:"amount"`       // sempre positivo
	BalanceAfter int       `json:"balance_after"`
	Source       string    `json:"source"`       // battle_tag | loot | death
	CreatedAt    time.Time `json:"created_at"`
}

// writeLedger persiste un movimento di oro in modo asincrono.
// Non blocca il turno: gli errori vengono loggati, non propagati.
func writeLedger(ctx context.Context, database db.DBClient, entry ledgerEntry) {
	go func() {
		query := fmt.Sprintf(
			`CREATE transaction_log CONTENT {
				character_id:  %q,
				turn_id:       %d,
				type:          %q,
				amount:        %d,
				balance_after: %d,
				source:        %q,
				created_at:    time::now()
			}`,
			entry.CharacterID,
			entry.TurnID,
			entry.Type,
			entry.Amount,
			entry.BalanceAfter,
			entry.Source,
		)
		if _, err := database.Query(query, nil); err != nil {
			slog.Warn("ledger write fallita", "char", entry.CharacterID, "err", err)
		}
	}()
}

// recordGoldDelta confronta il saldo prima e dopo un turno e scrive
// una voce nel ledger se l'oro è cambiato.
func recordGoldDelta(ctx context.Context, database db.DBClient, charID string, turnID, before, after int, source string) {
	delta := after - before
	if delta == 0 {
		return
	}

	entryType := "gain"
	amount := delta
	if delta < 0 {
		if source == "death" {
			entryType = "penalty"
		} else {
			entryType = "loss"
		}
		amount = -delta
	} else if source == "loot" {
		entryType = "loot"
	}

	writeLedger(ctx, database, ledgerEntry{
		CharacterID:  charID,
		TurnID:       turnID,
		Type:         entryType,
		Amount:       amount,
		BalanceAfter: after,
		Source:       source,
	})
}
