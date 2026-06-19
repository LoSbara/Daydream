package models

// Quest rappresenta una missione con ciclo di vita completo.
type Quest struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	GiverNPC    string           `json:"giver_npc,omitempty"`
	Category    string           `json:"category"`   // "main" | "side" | "urgent" | "world_event"
	Difficulty  int              `json:"difficulty"` // 1-5
	Urgency     string           `json:"urgency"`    // "low" | "medium" | "high" | "critical"
	Objectives  []QuestObjective `json:"objectives"`
	Rewards     QuestReward      `json:"rewards"`
	Status      string           `json:"status"` // "active" | "completed" | "failed" | "expired"
	StartedAt   int              `json:"started_at"`
	CompletedAt int              `json:"completed_at,omitempty"`
	// Tempo in-game
	DeadlineDay  int `json:"deadline_day"`  // giorno assoluto in-game della scadenza
	DeadlineHour int `json:"deadline_hour"` // ora assoluta in-game della scadenza
	// Escalation
	EscalationStage   int               `json:"escalation_stage"`            // stage corrente (0 = iniziale)
	Escalations       []QuestEscalation `json:"escalations"`                 // fasi di escalation
	ConsequenceOnFail string            `json:"consequence_on_fail,omitempty"`
}

// QuestEscalation è una fase di peggioramento della quest.
type QuestEscalation struct {
	Stage            int    `json:"stage"`
	Description      string `json:"description"`               // narrativo: cosa succede
	WorldFlagKey     string `json:"world_flag_key,omitempty"`  // flag emesso automaticamente
	WorldFlagValue   string `json:"world_flag_value,omitempty"`
	TriggerAtPercent int    `json:"trigger_at_percent"` // % del tempo usato (0-100)
}

// QuestObjective è un singolo obiettivo di una quest.
type QuestObjective struct {
	Description string `json:"description"`
	Current     int    `json:"current"`
	Required    int    `json:"required"`
	Done        bool   `json:"done"`
}

// QuestReward è la ricompensa al completamento di una quest.
type QuestReward struct {
	Gold  int    `json:"gold,omitempty"`
	Exp   int    `json:"exp,omitempty"`
	Items []Item `json:"items,omitempty"`
}

// QuestsStateUpdate contiene gli aggiornamenti alle quest emessi dal GM.
type QuestsStateUpdate struct {
	Start    *Quest               `json:"start,omitempty"`
	Complete string               `json:"complete,omitempty"`
	Fail     string               `json:"fail,omitempty"`
	Progress []QuestProgressUpdate `json:"progress,omitempty"`
}

// QuestProgressUpdate aggiorna il contatore di un obiettivo.
type QuestProgressUpdate struct {
	QuestID  string `json:"quest_id"`
	ObjIndex int    `json:"obj_index"`
	Delta    int    `json:"delta"`
}
