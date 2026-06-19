package models

type SpecializationOption struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Flavor      string         `json:"flavor"`
	StatBonus   map[string]int `json:"stat_bonus,omitempty"`
	PassiveDesc string         `json:"passive_desc"`
}

type SpecializationTier struct {
	Tier    int                    `json:"tier"`
	Level   int                    `json:"level"`
	Options []SpecializationOption `json:"options"`
}

type SpecChoiceAvailable struct {
	Tier    int                    `json:"tier"`
	Level   int                    `json:"level"`
	Options []SpecializationOption `json:"options"`
}
