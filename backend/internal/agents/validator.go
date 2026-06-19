package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"daydream/internal/db"
	"daydream/internal/models"
	"strings"
)

// ValidationScore è il risultato del Validator Agent per un singolo turno.
type ValidationScore struct {
	SessionID      string   `json:"session_id"`
	TurnID         int      `json:"turn_id"`
	Total          int      `json:"total"` // 0–100
	Issues         []string `json:"issues"`
	SuggestCompact bool     `json:"suggest_compact"`
}

// Validator valuta la qualità della risposta GM dopo ogni turno, in modo asincrono.
type Validator struct {
	db *db.Client
}

func NewValidator(database *db.Client) *Validator {
	return &Validator{db: database}
}

// ValidateAsync avvia la validazione in background. Non blocca il turno.
func (v *Validator) ValidateAsync(ctx context.Context, resp *models.GMResponse, state *models.FullState) {
	go func() {
		score := v.score(resp, state)
		if err := v.logScore(score); err != nil {
			log.Printf("[validator] log fallito (turn %d): %v", score.TurnID, err)
		}
		if score.Total < 40 {
			log.Printf("[validator] WARN score basso turn %d (%d/100): %v",
				score.TurnID, score.Total, score.Issues)
		}
	}()
}

const compactThresholdChars = 3200 // ≈800 token a 4 char/token

func (v *Validator) score(resp *models.GMResponse, state *models.FullState) ValidationScore {
	total := 100
	var issues []string
	suggestCompact := false

	// 1. narrative_length
	nLen := len(resp.Narrative)
	if nLen < 100 {
		total -= 25
		issues = append(issues, fmt.Sprintf("narrative corta: %d char (min 100)", nLen))
	} else if nLen > 2000 {
		total -= 10
		issues = append(issues, fmt.Sprintf("narrative lunga: %d char (max 2000)", nLen))
	}

	// 2. json_completeness: in combat senza battle_tags → penalità
	if state.Session.CombatActive && len(resp.BattleTags) == 0 {
		total -= 20
		issues = append(issues, "combat attivo ma battle_tags assenti")
	}

	// 3. money_in_state_updates: il GM non deve modificare i soldi direttamente
	if resp.StateUpdates != nil {
		raw, _ := json.Marshal(resp.StateUpdates)
		if strings.Contains(strings.ToLower(string(raw)), "money") {
			total -= 15
			issues = append(issues, "state_updates contiene 'money' (vietato: usare GOLD_GAIN/GOLD_LOSE)")
		}
	}

	// 4. context_memo_growth
	memoLen := len(state.Session.ContextMemo)
	if memoLen > compactThresholdChars {
		suggestCompact = true
		total -= 5
		issues = append(issues, fmt.Sprintf("context_memo %d char → compaction suggerita", memoLen))
	}

	// 5. battle_tags_plausibility: nemico presente ma nessun COMBAT_HIT_ENEMY → warning
	if state.Session.CombatActive && state.Session.CurrentEnemy != nil {
		hasHit := false
		for _, tag := range resp.BattleTags {
			if strings.HasPrefix(tag, "COMBAT_HIT_ENEMY") {
				hasHit = true
				break
			}
		}
		if !hasHit {
			total -= 10
			issues = append(issues, "nemico attivo ma nessun tag COMBAT_HIT_ENEMY")
		}
	}

	if total < 0 {
		total = 0
	}

	return ValidationScore{
		SessionID:      state.Session.ID,
		TurnID:         state.Session.TurnID,
		Total:          total,
		Issues:         issues,
		SuggestCompact: suggestCompact,
	}
}

func (v *Validator) logScore(score ValidationScore) error {
	if score.Issues == nil {
		score.Issues = []string{}
	}
	issuesJSON, _ := json.Marshal(score.Issues)
	sql := fmt.Sprintf(`CREATE validation_log CONTENT {
		session_id: %s,
		turn_id: %d,
		total: %d,
		issues: %s,
		suggest_compact: %v,
		created_at: time::now()
	}`,
		jsonStr(score.SessionID),
		score.TurnID,
		score.Total,
		string(issuesJSON),
		score.SuggestCompact,
	)
	return v.db.Exec(sql, nil)
}

func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
