package models

// Resource rappresenta una risorsa con valore corrente e massimo (HP, MP, STM).
type Resource struct {
	Current int `json:"current"`
	Max     int `json:"max"`
}

// Stats contiene tutte le statistiche del personaggio.
type Stats struct {
	HP  Resource `json:"HP"`
	MP  Resource `json:"MP"`
	STM Resource `json:"STM"`
	STR int      `json:"STR"`
	DEX int      `json:"DEX"`
	AGI int      `json:"AGI"`
	TEC int      `json:"TEC"`
	VIT int      `json:"VIT"`
	LUC int      `json:"LUC"`
}

// StatusEffect rappresenta un effetto di stato attivo (buff/debuff).
type StatusEffect struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Icon           string `json:"icon"`
	Type           string `json:"type"` // "buff" | "debuff"
	TurnsRemaining int    `json:"turns_remaining"`
	Value          int    `json:"value"`
	Color          string `json:"color"`
}

// Reputation contiene i valori di reputazione con le fazioni.
type Reputation struct {
	HuntersGuild int `json:"hunters_guild"`
	Merchants    int `json:"merchants"`
	CityGuard    int `json:"city_guard"`
	Scholars     int `json:"scholars"`
	Underground  int `json:"underground"`
}

// ActionCounters traccia le azioni del giocatore per lo sblocco di titoli e skill speciali.
type ActionCounters struct {
	EnemiesDefeated     int      `json:"enemies_defeated"`
	Dodges              int      `json:"dodges"`
	Criticals           int      `json:"criticals"`
	EnemiesAnalyzed     int      `json:"enemies_analyzed"`
	ZonesVisited        []string `json:"zones_visited"`
	MaxMoney            int      `json:"max_money"`
	EliteKills          int      `json:"elite_kills"`
	UniqueCompleted     int      `json:"unique_completed"`
	MaxSkillsInCombat   int      `json:"max_skills_in_combat"`
	NearDeathSurvives   int      `json:"near_death_survives"`
}

// Character è il profilo completo del personaggio giocante.
type Character struct {
	ID                  string            `json:"id"`
	UserID              string            `json:"user_id"`
	Name                string            `json:"name"`
	Job                 string            `json:"job"`
	Subclass            *string           `json:"subclass,omitempty"`
	AdvancedClass       *string           `json:"advanced_class,omitempty"`
	Level               int               `json:"level"`
	Experience          int               `json:"experience"`
	ExperienceToNext    int               `json:"experience_to_next"`
	Stats               Stats             `json:"stats"`
	StatPointsAvailable int               `json:"stat_points_available"`
	Money               int               `json:"money"`
	SkillSlots          int               `json:"skill_slots"`
	Titles              []string          `json:"titles"`
	StatusEffects       []StatusEffect    `json:"status_effects"`
	Reputation          Reputation        `json:"reputation"`
	Flags               map[string]any    `json:"flags"`
	ActionCounters      ActionCounters    `json:"action_counters"`
	SkillCooldowns      map[string]int    `json:"skill_cooldowns"`
	ChosenSpecs         []string          `json:"chosen_specs,omitempty"`
	TreePointsAvailable int               `json:"tree_points_available"`
	SkillTreeUnlocks    []string          `json:"skill_tree_unlocks,omitempty"`
	CustomSkills        []GMCustomSkill   `json:"custom_skills,omitempty"`
}

// Item rappresenta un oggetto nell'inventario.
type Item struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	Type               string         `json:"type"` // weapon | armor | consumable | material
	Slot               string         `json:"slot"`
	StatBonus          map[string]int `json:"stat_bonus"`
	Rarity             string         `json:"rarity"`
	Price              int            `json:"price"`
	Quantity           int            `json:"quantity"`
	Appraised          bool           `json:"appraised"`
	EnhancementLevel   int            `json:"enhancement_level"`
	PerceivedStatBonus map[string]int `json:"perceived_stat_bonus,omitempty"`
	PerceivedRarity    string         `json:"perceived_rarity,omitempty"`
	Analyzed           bool           `json:"analyzed,omitempty"` // per item di mercato
}

// EquippedSlots rappresenta gli slot di equipaggiamento attivi.
type EquippedSlots struct {
	Weapon     *Item `json:"weapon"`
	Offhand    *Item `json:"offhand"`
	Head       *Item `json:"head"`
	Chest      *Item `json:"chest"`
	Legs       *Item `json:"legs"`
	Boots      *Item `json:"boots"`
	Accessory1 *Item `json:"accessory_1"`
	Accessory2 *Item `json:"accessory_2"`
}

// StatBonuses sono i bonus stat aggregati dall'equipaggiamento.
type StatBonuses struct {
	STR      int `json:"STR"`
	DEX      int `json:"DEX"`
	AGI      int `json:"AGI"`
	TEC      int `json:"TEC"`
	VIT      int `json:"VIT"`
	LUC      int `json:"LUC"`
	HPBonus  int `json:"HP_bonus"`
	MPBonus  int `json:"MP_bonus"`
	STMBonus int `json:"STM_bonus"`
}

// Inventory contiene l'equipaggiamento e la borsa del personaggio.
type Inventory struct {
	ID                         string      `json:"id"`
	CharacterID                string      `json:"character_id"`
	Equipped                   EquippedSlots `json:"equipped"`
	StatBonusesFromEquipment   StatBonuses   `json:"stat_bonuses_from_equipment"`
	Bag                        []Item      `json:"bag"`
}

// TotalStat calcola il valore effettivo di una stat base + bonus equipaggiamento.
func TotalStat(char *Character, inv *Inventory, stat string) int {
	base := 0
	switch stat {
	case "STR": base = char.Stats.STR
	case "DEX": base = char.Stats.DEX
	case "AGI": base = char.Stats.AGI
	case "TEC": base = char.Stats.TEC
	case "VIT": base = char.Stats.VIT
	case "LUC": base = char.Stats.LUC
	}
	bonus := 0
	switch stat {
	case "STR": bonus = inv.StatBonusesFromEquipment.STR
	case "DEX": bonus = inv.StatBonusesFromEquipment.DEX
	case "AGI": bonus = inv.StatBonusesFromEquipment.AGI
	case "TEC": bonus = inv.StatBonusesFromEquipment.TEC
	case "VIT": bonus = inv.StatBonusesFromEquipment.VIT
	case "LUC": bonus = inv.StatBonusesFromEquipment.LUC
	}
	return base + bonus
}
