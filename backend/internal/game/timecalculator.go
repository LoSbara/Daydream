package game

import (
	"strings"

	"daydream/internal/models"
)

// timePerCategory: minuti base per categoria di azione.
var timePerCategory = map[string]int{
	"conversation":    25,
	"combat":          20,
	"exploration":     30,
	"travel_local":    20,
	"travel_regional": 240,
	"travel_long":     1440,
	"rest":            420,
	"crafting":        60,
}

// minTurnMinutes: ogni turno avanza il clock di almeno questo valore.
const minTurnMinutes = 10

// CalculateTimeElapsed calcola i minuti trascorsi in base alla action_category del GM
// e ai battle tags del turno.
func CalculateTimeElapsed(category string, tags []string) int {
	base, ok := timePerCategory[category]
	if !ok {
		base = inferTimeFromTags(tags)
	}

	switch category {
	case "combat":
		enemies := countTagPrefix(tags, "ENEMY_DEAD")
		base += enemies * 15
	case "exploration":
		moves := countTagPrefix(tags, "DUNGEON_MOVE_")
		if moves > 0 {
			base = moves * 20
		}
	}

	if base < minTurnMinutes {
		base = minTurnMinutes
	}
	return base
}

// inferTimeFromTags: stima il tempo dai battle tag se manca la categoria.
func inferTimeFromTags(tags []string) int {
	if countTagPrefix(tags, "ENEMY_DEAD") > 0 || countTagPrefix(tags, "ENEMY_HP_") > 0 {
		return 30
	}
	if countTagPrefix(tags, "DUNGEON_MOVE_") > 0 {
		return 25
	}
	if countTagPrefix(tags, "PLAYER_HP_+") > 0 && countTagPrefix(tags, "ENEMY_HP_") == 0 {
		return 120
	}
	return 20
}

// AdvanceClock aggiorna il GameTime e l'HoursAwake sulla sessione.
// Se l'azione è "rest", azzera HoursAwake.
func AdvanceClock(sess *models.GameSession, minutes int, category string) {
	sess.GameTime = sess.GameTime.AddMinutes(minutes)
	if category == "rest" {
		sess.HoursAwake = 0
	} else {
		sess.HoursAwake += float64(minutes) / 60.0
	}
}

// SleepDeprivationDebuffs restituisce i debuff da applicare in base alle ore senza dormire.
func SleepDeprivationDebuffs(hoursAwake float64) []models.StatusEffect {
	switch {
	case hoursAwake >= 48:
		return []models.StatusEffect{{
			ID:             "sleep_deprivation",
			Name:           "Collasso imminente",
			Icon:           "💤",
			Type:           "debuff",
			TurnsRemaining: 999,
			Value:          10,
			Color:          "#ff0000",
		}}
	case hoursAwake >= 36:
		return []models.StatusEffect{{
			ID:             "sleep_deprivation",
			Name:           "Esausto",
			Icon:           "😩",
			Type:           "debuff",
			TurnsRemaining: 999,
			Value:          5,
			Color:          "#ff6b6b",
		}}
	case hoursAwake >= 20:
		return []models.StatusEffect{{
			ID:             "sleep_deprivation",
			Name:           "Stanco",
			Icon:           "😴",
			Type:           "debuff",
			TurnsRemaining: 999,
			Value:          2,
			Color:          "#ffaa00",
		}}
	default:
		return nil
	}
}

// ApplySleepDeprivation aggiorna i debuff da sonno sul personaggio.
func ApplySleepDeprivation(char *models.Character, hoursAwake float64) {
	filtered := char.StatusEffects[:0]
	for _, se := range char.StatusEffects {
		if se.Name != "Stanco" && se.Name != "Esausto" && se.Name != "Collasso imminente" {
			filtered = append(filtered, se)
		}
	}
	char.StatusEffects = filtered

	debuffs := SleepDeprivationDebuffs(hoursAwake)
	char.StatusEffects = append(char.StatusEffects, debuffs...)
}

// countTagPrefix conta quanti tag iniziano con il prefisso dato.
func countTagPrefix(tags []string, prefix string) int {
	n := 0
	for _, t := range tags {
		if strings.HasPrefix(t, prefix) {
			n++
		}
	}
	return n
}
