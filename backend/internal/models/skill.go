package models

// Skill rappresenta una skill del catalogo (designer-authored, non AI-generated).
type Skill struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	Job              string         `json:"job"`
	Tier             int            `json:"tier"`
	MPCost           int            `json:"mp_cost"`
	STMCost          int            `json:"stm_cost"`
	CooldownTurns    int            `json:"cooldown_turns"`
	Tags             []string       `json:"tags"`           // battle tags pre-applicati al GM
	DamageMultiplier float64        `json:"damage_multiplier"`
	HealAmount       int            `json:"heal_amount,omitempty"`
	Element          string         `json:"element,omitempty"`
	UnlockCondition  map[string]any `json:"unlock_condition,omitempty"`
}

// SkillUseResult contiene il risultato dell'uso di una skill.
type SkillUseResult struct {
	Skill     *Skill
	ExtraTags []string // tag aggiuntivi calcolati server-side (es. danno effettivo)
	Denied    bool
	DenyReason string
}
