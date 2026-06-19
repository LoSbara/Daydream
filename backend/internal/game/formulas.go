package game

import (
	"math"
	"math/rand"
	"daydream/internal/models"
)

// DamageResult contiene il risultato di un calcolo danno.
type DamageResult struct {
	Raw      int
	Final    int
	IsCrit   bool
	IsDodge  bool
}

// PhysicalDamage calcola il danno fisico del giocatore su un nemico.
// Dipende da STR + eventuale bonus arma.
func PhysicalDamage(char *models.Character, inv *models.Inventory, enemyDEF int) DamageResult {
	str := models.TotalStat(char, inv, "STR")
	luc := models.TotalStat(char, inv, "LUC")

	rawDmg := int(math.Round(float64(str) * 1.5))
	critChance := float64(luc) / 200.0 // LUC 10 → 5% crit
	isCrit := rand.Float64() < critChance

	finalDmg := rawDmg - enemyDEF/2
	if finalDmg < 1 {
		finalDmg = 1
	}
	if isCrit {
		finalDmg = int(math.Round(float64(finalDmg) * 1.5))
	}

	return DamageResult{Raw: rawDmg, Final: finalDmg, IsCrit: isCrit}
}

// DodgeChance calcola la probabilità di schivata del giocatore.
func DodgeChance(char *models.Character, inv *models.Inventory) float64 {
	agi := models.TotalStat(char, inv, "AGI")
	// AGI 10 → 5%, AGI 20 → ~9.5%
	return 1 - math.Pow(0.995, float64(agi))
}

// RollDodge restituisce true se il giocatore schiva l'attacco.
func RollDodge(char *models.Character, inv *models.Inventory) bool {
	return rand.Float64() < DodgeChance(char, inv)
}

// EnemyDamage calcola il danno che un nemico infligge al giocatore.
func EnemyDamage(enemyATK int, char *models.Character, inv *models.Inventory) DamageResult {
	agi := models.TotalStat(char, inv, "AGI")
	isDodge := rand.Float64() < DodgeChance(char, inv)
	if isDodge {
		return DamageResult{IsDodge: true}
	}

	_ = agi // future: bonus difesa per AGI
	finalDmg := enemyATK
	if finalDmg < 1 {
		finalDmg = 1
	}
	return DamageResult{Raw: enemyATK, Final: finalDmg}
}

// ExperienceToNextLevel restituisce l'esperienza necessaria per passare al livello successivo.
func ExperienceToNextLevel(level int) int {
	// Curva moderatamente crescente: 100, 150, 225, 337, ...
	return int(math.Round(100.0 * math.Pow(1.5, float64(level-1))))
}

// Clamp limita un valore tra min e max.
func Clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ComputeMaxHP calcola il max HP sulla base delle stats.
func ComputeMaxHP(char *models.Character, inv *models.Inventory) int {
	vit := models.TotalStat(char, inv, "VIT")
	return 100 + (vit-10)*5 + inv.StatBonusesFromEquipment.HPBonus
}

// ComputeMaxMP calcola il max MP sulla base delle stats.
func ComputeMaxMP(char *models.Character, inv *models.Inventory) int {
	tec := models.TotalStat(char, inv, "TEC")
	return 50 + (tec-10)*3 + inv.StatBonusesFromEquipment.MPBonus
}

// ComputeMaxSTM calcola il max STM sulla base delle stats.
func ComputeMaxSTM(char *models.Character, inv *models.Inventory) int {
	return 100 + inv.StatBonusesFromEquipment.STMBonus
}
