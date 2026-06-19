package game

import (
	"testing"

	"daydream/internal/models"
)

func baseQuest(diff int, urgency string) models.Quest {
	return models.Quest{
		ID:         "test_quest",
		Difficulty: diff,
		Urgency:    urgency,
		Rewards: models.QuestReward{
			Gold: 9999,
			Exp:  9999,
		},
	}
}

func baseTime() models.GameTime {
	return models.GameTime{Day: 1, Hour: 8, Minute: 0}
}

// ── Gold cap ──────────────────────────────────────────────────────────────────

func TestBalanceQuest_GoldCappedByDiff(t *testing.T) {
	q := baseQuest(1, "medium")
	charLevel := 5
	BalanceQuest(&q, charLevel, baseTime())
	// charLevel=5, diff=1 (mult=0.5): maxGold = 5*25*0.5 = 62
	maxGold := int(float64(charLevel) * 25.0 * 0.5)
	if q.Rewards.Gold > maxGold {
		t.Errorf("Gold %d supera il cap %d per diff=1, level=5", q.Rewards.Gold, maxGold)
	}
}

func TestBalanceQuest_GoldHigherForHigherDiff(t *testing.T) {
	q1 := baseQuest(1, "medium")
	q5 := baseQuest(5, "medium")
	BalanceQuest(&q1, 10, baseTime())
	BalanceQuest(&q5, 10, baseTime())
	if q1.Rewards.Gold >= q5.Rewards.Gold {
		t.Errorf("gold diff=1 (%d) >= gold diff=5 (%d), ci si aspetta diff=5 maggiore",
			q1.Rewards.Gold, q5.Rewards.Gold)
	}
}

// ── Exp cap ───────────────────────────────────────────────────────────────────

func TestBalanceQuest_ExpCappedByDiff(t *testing.T) {
	q := baseQuest(2, "medium")
	BalanceQuest(&q, 3, baseTime())
	maxExp := int(float64(3) * 40.0 * 1.0) // diff=2 mult=1.0
	if q.Rewards.Exp > maxExp {
		t.Errorf("Exp %d supera il cap %d", q.Rewards.Exp, maxExp)
	}
}

// ── Difficulty clamp ──────────────────────────────────────────────────────────

func TestBalanceQuest_DiffClampedTo1(t *testing.T) {
	q := baseQuest(0, "medium")
	before := q.Rewards.Gold
	charLevel := 5
	BalanceQuest(&q, charLevel, baseTime())
	// Con diff clamped a 1, il cap è applicato e gold deve essere <= before
	maxGold := int(float64(charLevel) * 25.0 * 0.5)
	if q.Rewards.Gold > before && q.Rewards.Gold > maxGold {
		t.Errorf("diff=0 non clampato: gold %d troppo alto", q.Rewards.Gold)
	}
}

func TestBalanceQuest_DiffClampedTo5(t *testing.T) {
	q := baseQuest(99, "medium")
	charLevel := 5
	BalanceQuest(&q, charLevel, baseTime())
	maxGold := int(float64(charLevel) * 25.0 * 8.0) // diff=5, mult=8.0
	if q.Rewards.Gold > maxGold {
		t.Errorf("diff=99 non clampato a 5: gold %d > %d", q.Rewards.Gold, maxGold)
	}
}

// ── Deadline ─────────────────────────────────────────────────────────────────

func TestBalanceQuest_DeadlineSet(t *testing.T) {
	q := baseQuest(3, "high")
	BalanceQuest(&q, 5, baseTime())
	// Con urgency=high il deadline deve essere nel futuro
	start := baseTime().TotalMinutes()
	deadline := models.GameTime{Day: q.DeadlineDay, Hour: q.DeadlineHour}.TotalMinutes()
	if deadline <= start {
		t.Errorf("deadline (giorno %d, ora %d) non è nel futuro rispetto all'inizio",
			q.DeadlineDay, q.DeadlineHour)
	}
}

func TestBalanceQuest_CriticalUrgencyDeadlineShorterThanLow(t *testing.T) {
	qLow := baseQuest(3, "low")
	qCritical := baseQuest(3, "critical")
	BalanceQuest(&qLow, 5, baseTime())
	BalanceQuest(&qCritical, 5, baseTime())

	deadlineLow := models.GameTime{Day: qLow.DeadlineDay, Hour: qLow.DeadlineHour}.TotalMinutes()
	deadlineCritical := models.GameTime{Day: qCritical.DeadlineDay, Hour: qCritical.DeadlineHour}.TotalMinutes()

	if deadlineCritical >= deadlineLow {
		t.Errorf("urgency=critical (%d min) non è più breve di urgency=low (%d min)",
			deadlineCritical, deadlineLow)
	}
}

// ── Escalations ───────────────────────────────────────────────────────────────

func TestBalanceQuest_DefaultEscalationsGenerated(t *testing.T) {
	q := baseQuest(3, "medium")
	// Nessun escalation predefinito
	BalanceQuest(&q, 5, baseTime())
	if len(q.Escalations) == 0 {
		t.Error("nessun escalation generato automaticamente")
	}
}

func TestBalanceQuest_ExistingEscalationsStageNumbered(t *testing.T) {
	q := baseQuest(3, "medium")
	q.Escalations = []models.QuestEscalation{
		{Description: "fase 1"},
		{Description: "fase 2"},
	}
	BalanceQuest(&q, 5, baseTime())
	for i, esc := range q.Escalations {
		if esc.Stage != i+1 {
			t.Errorf("escalation[%d].Stage = %d, atteso %d", i, esc.Stage, i+1)
		}
	}
}

func TestBalanceQuest_NilQuestNocrash(t *testing.T) {
	// Non deve crashare
	BalanceQuest(nil, 5, baseTime())
}
