package game

import (
	"testing"

	"daydream/internal/models"
)

// ── Clamp ─────────────────────────────────────────────────────────────────────

func TestClamp_WithinRange(t *testing.T) {
	if got := Clamp(50, 0, 100); got != 50 {
		t.Errorf("Clamp(50, 0, 100) = %d, atteso 50", got)
	}
}

func TestClamp_BelowMin(t *testing.T) {
	if got := Clamp(-10, 0, 100); got != 0 {
		t.Errorf("Clamp(-10, 0, 100) = %d, atteso 0", got)
	}
}

func TestClamp_AboveMax(t *testing.T) {
	if got := Clamp(200, 0, 100); got != 100 {
		t.Errorf("Clamp(200, 0, 100) = %d, atteso 100", got)
	}
}

func TestClamp_AtBoundaries(t *testing.T) {
	if got := Clamp(0, 0, 100); got != 0 {
		t.Errorf("Clamp(0, 0, 100) = %d, atteso 0", got)
	}
	if got := Clamp(100, 0, 100); got != 100 {
		t.Errorf("Clamp(100, 0, 100) = %d, atteso 100", got)
	}
}

func TestClamp_SameMinMax(t *testing.T) {
	if got := Clamp(50, 10, 10); got != 10 {
		t.Errorf("Clamp(50, 10, 10) = %d, atteso 10", got)
	}
}

// ── ExperienceToNextLevel ─────────────────────────────────────────────────────

func TestExperienceToNextLevel_Level1(t *testing.T) {
	// Livello 1: 100 * 1.5^0 = 100
	if got := ExperienceToNextLevel(1); got != 100 {
		t.Errorf("ExperienceToNextLevel(1) = %d, atteso 100", got)
	}
}

func TestExperienceToNextLevel_Level2(t *testing.T) {
	// Livello 2: 100 * 1.5^1 = 150
	if got := ExperienceToNextLevel(2); got != 150 {
		t.Errorf("ExperienceToNextLevel(2) = %d, atteso 150", got)
	}
}

func TestExperienceToNextLevel_Level3(t *testing.T) {
	// Livello 3: 100 * 1.5^2 = 225
	if got := ExperienceToNextLevel(3); got != 225 {
		t.Errorf("ExperienceToNextLevel(3) = %d, atteso 225", got)
	}
}

func TestExperienceToNextLevel_Monotonic(t *testing.T) {
	// I valori devono crescere monotonicamente
	prev := ExperienceToNextLevel(1)
	for lv := 2; lv <= 20; lv++ {
		cur := ExperienceToNextLevel(lv)
		if cur <= prev {
			t.Errorf("livello %d: exp %d non maggiore di livello %d: exp %d", lv, cur, lv-1, prev)
		}
		prev = cur
	}
}

// ── ComputeMaxHP / MP / STM ───────────────────────────────────────────────────

func baseChar(vit, tec int) *models.Character {
	return &models.Character{
		Stats: models.Stats{VIT: vit, TEC: tec},
	}
}

func baseInv() *models.Inventory {
	return &models.Inventory{StatBonusesFromEquipment: models.StatBonuses{}}
}

func TestComputeMaxHP_BaseVit10(t *testing.T) {
	// VIT=10 → 100 + (10-10)*5 = 100
	got := ComputeMaxHP(baseChar(10, 10), baseInv())
	if got != 100 {
		t.Errorf("ComputeMaxHP VIT=10 = %d, atteso 100", got)
	}
}

func TestComputeMaxHP_HighVit(t *testing.T) {
	// VIT=20 → 100 + (20-10)*5 = 150
	got := ComputeMaxHP(baseChar(20, 10), baseInv())
	if got != 150 {
		t.Errorf("ComputeMaxHP VIT=20 = %d, atteso 150", got)
	}
}

func TestComputeMaxMP_BaseTec10(t *testing.T) {
	// TEC=10 → 50 + (10-10)*3 = 50
	got := ComputeMaxMP(baseChar(10, 10), baseInv())
	if got != 50 {
		t.Errorf("ComputeMaxMP TEC=10 = %d, atteso 50", got)
	}
}

func TestComputeMaxMP_HighTec(t *testing.T) {
	// TEC=20 → 50 + (20-10)*3 = 80
	got := ComputeMaxMP(baseChar(10, 20), baseInv())
	if got != 80 {
		t.Errorf("ComputeMaxMP TEC=20 = %d, atteso 80", got)
	}
}

func TestComputeMaxSTM_Base(t *testing.T) {
	got := ComputeMaxSTM(baseChar(10, 10), baseInv())
	if got != 100 {
		t.Errorf("ComputeMaxSTM base = %d, atteso 100", got)
	}
}

func TestComputeMaxSTM_WithBonus(t *testing.T) {
	inv := &models.Inventory{StatBonusesFromEquipment: models.StatBonuses{STMBonus: 30}}
	got := ComputeMaxSTM(baseChar(10, 10), inv)
	if got != 130 {
		t.Errorf("ComputeMaxSTM con bonus 30 = %d, atteso 130", got)
	}
}
