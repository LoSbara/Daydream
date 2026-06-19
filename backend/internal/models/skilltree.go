package models

type SkillTreeNode struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Type          string         `json:"type"`     // "active" | "passive"
	Branch        string         `json:"branch"`
	BranchOrder   int            `json:"branch_order"` // 1-4
	Prerequisites []string       `json:"prerequisites,omitempty"`
	SpecRequired  string         `json:"spec_required,omitempty"`
	Cost          int            `json:"cost"` // costo in punti skill (basato sul tier: 3/7/15/28)
	MPCost        int            `json:"mp_cost,omitempty"`
	STMCost       int            `json:"stm_cost,omitempty"`
	Cooldown      int            `json:"cooldown,omitempty"` // turni
	StatBonus     map[string]int `json:"stat_bonus,omitempty"`
	EffectDesc    string         `json:"effect_desc"`
	Unlocked      bool           `json:"unlocked"` // calcolato runtime, non DB
	Available     bool           `json:"available"` // prerequisiti soddisfatti
}

type SkillBranch struct {
	Name   string          `json:"name"`
	Icon   string          `json:"icon"`
	Skills []SkillTreeNode `json:"skills"`
}

type ClassSkillTree struct {
	Class    string        `json:"class"`
	Branches []SkillBranch `json:"branches"`
}
