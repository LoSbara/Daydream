package game

import (
	"daydream/internal/models"
)

// diffMult: moltiplicatori ricompense per difficoltà.
var diffMult = map[int]float64{
	1: 0.5,
	2: 1.0,
	3: 2.0,
	4: 4.0,
	5: 8.0,
}

// maxItemRarityByDiff: rarità massima consentita per difficoltà.
var maxItemRarityByDiff = map[int]string{
	1: "common",
	2: "uncommon",
	3: "rare",
	4: "epic",
	5: "legendary",
}

// urgencyBaseMinutes: minuti base per stage per urgency.
var urgencyBaseMinutes = map[string]int{
	"low":      25 * 60,
	"medium":   14 * 60,
	"high":     7 * 60,
	"critical": 3 * 60,
}

// diffTimeMultiplier: quest difficili danno più tempo.
var diffTimeMultiplier = map[int]float64{
	1: 0.8,
	2: 0.9,
	3: 1.0,
	4: 1.2,
	5: 1.4,
}

// BalanceQuest calcola e applica rewards bilanciate e deadline al GameTime corrente.
func BalanceQuest(quest *models.Quest, charLevel int, currentTime models.GameTime) {
	if quest == nil {
		return
	}

	diff := quest.Difficulty
	if diff < 1 {
		diff = 1
	}
	if diff > 5 {
		diff = 5
	}

	mult := diffMult[diff]

	// --- Rewards ---
	maxGold := int(float64(charLevel) * 25.0 * mult)
	if quest.Rewards.Gold > maxGold {
		quest.Rewards.Gold = maxGold
	}
	minGold := int(float64(charLevel) * 5.0 * mult)
	if diff >= 2 && quest.Rewards.Gold < minGold {
		quest.Rewards.Gold = minGold
	}

	maxExp := int(float64(charLevel) * 40.0 * mult)
	if quest.Rewards.Exp > maxExp {
		quest.Rewards.Exp = maxExp
	}
	minExp := int(float64(charLevel) * 10.0 * mult)
	if diff >= 2 && quest.Rewards.Exp < minExp {
		quest.Rewards.Exp = minExp
	}

	maxRarity := maxItemRarityByDiff[diff]
	rarityOrder := map[string]int{"common": 0, "uncommon": 1, "rare": 2, "epic": 3, "legendary": 4}
	for i := range quest.Rewards.Items {
		if rarityOrder[quest.Rewards.Items[i].Rarity] > rarityOrder[maxRarity] {
			quest.Rewards.Items[i].Rarity = maxRarity
		}
	}

	// --- Deadline ---
	urgency := quest.Urgency
	if urgency == "" {
		urgency = "medium"
	}
	baseMin, ok := urgencyBaseMinutes[urgency]
	if !ok {
		baseMin = urgencyBaseMinutes["medium"]
	}

	stageCount := len(quest.Escalations)
	if stageCount < 2 {
		stageCount = 2
	}

	totalMin := int(float64(baseMin*stageCount) * diffTimeMultiplier[diff])
	deadline := currentTime.AddMinutes(totalMin)
	quest.DeadlineDay = deadline.Day
	quest.DeadlineHour = deadline.Hour

	// --- Escalation auto-percentuali ---
	for i := range quest.Escalations {
		if quest.Escalations[i].TriggerAtPercent == 0 {
			quest.Escalations[i].TriggerAtPercent = (i + 1) * (100 / (len(quest.Escalations) + 1))
		}
		quest.Escalations[i].Stage = i + 1
	}

	if len(quest.Escalations) == 0 {
		quest.Escalations = defaultEscalations(quest.ID, urgency)
	}
}

// defaultEscalations genera stage di escalation generici.
func defaultEscalations(questID, urgency string) []models.QuestEscalation {
	_ = urgency
	return []models.QuestEscalation{
		{
			Stage:            1,
			Description:      "La situazione inizia a peggiorare.",
			WorldFlagKey:     "quest_" + questID + "_stage",
			WorldFlagValue:   "1",
			TriggerAtPercent: 33,
		},
		{
			Stage:            2,
			Description:      "La situazione è critica. Il tempo sta per scadere.",
			WorldFlagKey:     "quest_" + questID + "_stage",
			WorldFlagValue:   "2",
			TriggerAtPercent: 66,
		},
		{
			Stage:            3,
			Description:      "Ultima possibilità. Le conseguenze sono inevitabili se non si agisce ora.",
			WorldFlagKey:     "quest_" + questID + "_stage",
			WorldFlagValue:   "3",
			TriggerAtPercent: 90,
		},
	}
}
