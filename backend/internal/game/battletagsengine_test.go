package game

import (
	"testing"

	"daydream/internal/models"
)

// minimalState costruisce uno stato di gioco minimale per i test.
func minimalState() *models.FullState {
	return &models.FullState{
		Character: &models.Character{
			Money: 1000,
			Stats: models.Stats{
				HP:  models.Resource{Max: 100, Current: 100},
				MP:  models.Resource{Max: 80, Current: 80},
				STM: models.Resource{Max: 60, Current: 60},
			},
			Experience:       0,
			ExperienceToNext: 100,
			Level:            1,
			SkillCooldowns:   map[string]int{},
			ActionCounters:   models.ActionCounters{ZonesVisited: []string{}},
		},
		Inventory: &models.Inventory{
			Bag: []models.Item{},
		},
		Session: &models.GameSession{
			TurnID: 5,
		},
	}
}

// ── GOLD ─────────────────────────────────────────────────────────────────────

func TestApplyBattleTags_GoldGain(t *testing.T) {
	state := minimalState()
	ApplyBattleTags([]string{"GOLD_GAIN_150"}, state, nil)
	if state.Character.Money != 1150 {
		t.Errorf("money = %d, atteso 1150", state.Character.Money)
	}
}

func TestApplyBattleTags_GoldLose(t *testing.T) {
	state := minimalState()
	ApplyBattleTags([]string{"GOLD_LOSE_200"}, state, nil)
	if state.Character.Money != 800 {
		t.Errorf("money = %d, atteso 800", state.Character.Money)
	}
}

func TestApplyBattleTags_GoldLoseFloorZero(t *testing.T) {
	state := minimalState()
	state.Character.Money = 50
	ApplyBattleTags([]string{"GOLD_LOSE_500"}, state, nil)
	if state.Character.Money != 0 {
		t.Errorf("money = %d, atteso 0 (floor a zero)", state.Character.Money)
	}
}

func TestApplyBattleTags_GoldLoseCooldown(t *testing.T) {
	state := minimalState()
	state.Session.TurnID = 5
	state.Session.GoldLoseCooldownUntil = 7 // cooldown attivo
	ApplyBattleTags([]string{"GOLD_LOSE_300"}, state, nil)
	// Deve essere bloccato dal cooldown
	if state.Character.Money != 1000 {
		t.Errorf("money = %d dopo GOLD_LOSE bloccato, atteso 1000", state.Character.Money)
	}
}

func TestApplyBattleTags_GoldLoseCooldownExpired(t *testing.T) {
	state := minimalState()
	state.Session.TurnID = 10
	state.Session.GoldLoseCooldownUntil = 7 // cooldown scaduto
	ApplyBattleTags([]string{"GOLD_LOSE_100"}, state, nil)
	if state.Character.Money != 900 {
		t.Errorf("money = %d dopo GOLD_LOSE con cooldown scaduto, atteso 900", state.Character.Money)
	}
	// Cooldown deve essere rinnovato
	if state.Session.GoldLoseCooldownUntil != 12 {
		t.Errorf("cooldown_until = %d, atteso 12", state.Session.GoldLoseCooldownUntil)
	}
}

// ── HP / MP / STM ─────────────────────────────────────────────────────────────

func TestApplyBattleTags_PlayerHPDamage(t *testing.T) {
	state := minimalState()
	ApplyBattleTags([]string{"PLAYER_HP_-30"}, state, nil)
	if state.Character.Stats.HP.Current != 70 {
		t.Errorf("HP = %d, atteso 70", state.Character.Stats.HP.Current)
	}
}

func TestApplyBattleTags_PlayerHPHeal(t *testing.T) {
	state := minimalState()
	state.Character.Stats.HP.Current = 50
	ApplyBattleTags([]string{"PLAYER_HP_+20"}, state, nil)
	if state.Character.Stats.HP.Current != 70 {
		t.Errorf("HP = %d, atteso 70", state.Character.Stats.HP.Current)
	}
}

func TestApplyBattleTags_PlayerHPCapAtMax(t *testing.T) {
	state := minimalState()
	state.Character.Stats.HP.Current = 90
	ApplyBattleTags([]string{"PLAYER_HP_+50"}, state, nil)
	if state.Character.Stats.HP.Current != 100 {
		t.Errorf("HP = %d, atteso 100 (capped al max)", state.Character.Stats.HP.Current)
	}
}

func TestApplyBattleTags_PlayerHPFloorZero(t *testing.T) {
	state := minimalState()
	state.Character.Stats.HP.Current = 10
	events := ApplyBattleTags([]string{"PLAYER_HP_-200"}, state, nil)
	if state.Character.Stats.HP.Current != 0 {
		t.Errorf("HP = %d, atteso 0", state.Character.Stats.HP.Current)
	}
	_ = events // non verifichiamo PLAYER_DEAD automatico qui
}

// ── EXP / LEVEL UP ────────────────────────────────────────────────────────────

func TestApplyBattleTags_ExpGain(t *testing.T) {
	state := minimalState()
	ApplyBattleTags([]string{"EXP_GAIN_50"}, state, nil)
	if state.Character.Experience != 50 {
		t.Errorf("experience = %d, atteso 50", state.Character.Experience)
	}
}

func TestApplyBattleTags_ExpGainLevelUp(t *testing.T) {
	state := minimalState()
	state.Character.Experience = 80
	state.Character.ExperienceToNext = 100
	events := ApplyBattleTags([]string{"EXP_GAIN_50"}, state, nil)

	hasLevelUp := false
	for _, ev := range events {
		if ev.Type == "level_up" {
			hasLevelUp = true
		}
	}
	if !hasLevelUp {
		t.Errorf("atteso evento level_up ma non trovato; eventi: %+v", events)
	}
	if state.Character.Level != 2 {
		t.Errorf("livello = %d, atteso 2", state.Character.Level)
	}
}

// ── NEAR DEATH ────────────────────────────────────────────────────────────────

func TestApplyBattleTags_NearDeathEvent(t *testing.T) {
	state := minimalState()
	// HP = 20% del max = 20 → trigger near_death
	state.Character.Stats.HP.Current = 20
	events := ApplyBattleTags([]string{}, state, nil)

	hasNearDeath := false
	for _, ev := range events {
		if ev.Type == "near_death" {
			hasNearDeath = true
		}
	}
	if !hasNearDeath {
		t.Error("atteso evento near_death con HP al 20%")
	}
}

func TestApplyBattleTags_NoNearDeathAboveThreshold(t *testing.T) {
	state := minimalState()
	state.Character.Stats.HP.Current = 25 // 25% > 20%
	events := ApplyBattleTags([]string{}, state, nil)
	for _, ev := range events {
		if ev.Type == "near_death" {
			t.Errorf("near_death non atteso con HP al 25%%")
		}
	}
}

// ── PLAYER DEAD ───────────────────────────────────────────────────────────────

func TestApplyBattleTags_PlayerDead(t *testing.T) {
	state := minimalState()
	state.Character.Stats.HP.Current = 0
	events := ApplyBattleTags([]string{"PLAYER_DEAD"}, state, nil)

	hasDead := false
	for _, ev := range events {
		if ev.Type == "player_dead" {
			hasDead = true
		}
	}
	if !hasDead {
		t.Error("atteso evento player_dead")
	}
}

// ── LOOT DROP ────────────────────────────────────────────────────────────────

func TestApplyBattleTags_LootDrop(t *testing.T) {
	state := minimalState()
	events := ApplyBattleTags([]string{"LOOT_DROP"}, state, nil)
	if len(events) == 0 || events[0].Type != "loot_drop" {
		t.Errorf("atteso evento loot_drop, ricevuto: %+v", events)
	}
}

// ── CONTATORI ────────────────────────────────────────────────────────────────

func TestApplyBattleTags_Dodge(t *testing.T) {
	state := minimalState()
	ApplyBattleTags([]string{"PLAYER_DODGE"}, state, nil)
	if state.Character.ActionCounters.Dodges != 1 {
		t.Errorf("Dodges = %d, atteso 1", state.Character.ActionCounters.Dodges)
	}
}

func TestApplyBattleTags_Crit(t *testing.T) {
	state := minimalState()
	ApplyBattleTags([]string{"PLAYER_CRIT"}, state, nil)
	if state.Character.ActionCounters.Criticals != 1 {
		t.Errorf("Criticals = %d, atteso 1", state.Character.ActionCounters.Criticals)
	}
}

// ── SKILL_USE con nil registry ────────────────────────────────────────────────

func TestApplyBattleTags_SkillUseNilRegistryNocrash(t *testing.T) {
	// Deve passare senza panic quando skills è nil
	state := minimalState()
	ApplyBattleTags([]string{"SKILL_USE_fireball"}, state, nil)
}
