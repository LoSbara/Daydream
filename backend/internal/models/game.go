package models

// GameStateEnum rappresenta lo stato corrente del flusso di gioco.
type GameStateEnum string

const (
	StateWorldNavigation GameStateEnum = "WORLD_NAVIGATION"
	StateCombat          GameStateEnum = "COMBAT"
	StateDungeonExplore  GameStateEnum = "DUNGEON_EXPLORATION"
	StateDungeonCombat   GameStateEnum = "DUNGEON_COMBAT"
)

// Enemy rappresenta il nemico corrente in combattimento.
type Enemy struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Tier         string         `json:"tier"`
	Level        int            `json:"level"`
	HP           int            `json:"hp"`
	MaxHP        int            `json:"max_hp"`
	Stats        map[string]int `json:"stats"`
	Weaknesses   []string       `json:"weaknesses"`
	Resistances  []string       `json:"resistances"`
	CurrentPhase int            `json:"current_phase"`
}

// SessionMessage è un singolo messaggio nella history verbatim.
type SessionMessage struct {
	Role    string `json:"role"` // "player" | "gm"
	Content string `json:"content"`
}

// DungeonRoom è una singola stanza del dungeon.
type DungeonRoom struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Exits       map[string]string `json:"exits"`       // direzione → room_id ("nord", "sud", "est", "ovest")
	HasEnemy    bool              `json:"has_enemy"`
	EnemyTier   string            `json:"enemy_tier,omitempty"` // "normal" | "elite" | "boss"
	Visited     bool              `json:"visited"`
	IsEntrance  bool              `json:"is_entrance"`
	IsBoss      bool              `json:"is_boss"`
	Cleared     bool              `json:"cleared"` // stanza senza più minacce
}

// ActiveDungeon rappresenta il dungeon in cui si trova il personaggio.
type ActiveDungeon struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Difficulty  int                    `json:"difficulty"` // 1-5
	CurrentRoom string                 `json:"current_room"`
	Rooms       map[string]DungeonRoom `json:"rooms"`
	EnteredAt   int                    `json:"entered_at"` // turn_id al momento dell'ingresso
}

// GameSession è lo stato di gioco corrente del personaggio.
type GameSession struct {
	ID                     string          `json:"id"`
	CharacterID            string          `json:"character_id"`
	Location               string          `json:"location"`
	SubLocation            string          `json:"sub_location"`
	ZoneType               string          `json:"zone_type"` // safe_zone | combat_zone | dungeon
	GameState              GameStateEnum   `json:"game_state"`
	CombatActive           bool            `json:"combat_active"`
	CurrentEnemy           *Enemy          `json:"current_enemy,omitempty"`
	TacticalTension        int             `json:"tactical_tension"`
	SkillLoadout           []string        `json:"skill_loadout"`
	SessionLog             []SessionMessage `json:"session_log"`
	ContextMemo            string          `json:"context_memo"`
	PendingNarrativeEvents []string        `json:"pending_narrative_events"`
	QuestsActive           []Quest         `json:"quests_active"`
	QuestsCompleted        []Quest         `json:"quests_completed"`
	TurnID                 int             `json:"turn_id"`
	ActiveDungeon          *ActiveDungeon  `json:"active_dungeon,omitempty"`
	// Anti-duplicazione acquisti (3 layer)
	GoldLoseCooldownUntil int             `json:"gold_lose_cooldown_until"` // turno fino al quale GOLD_LOSE è bloccato
	LastGoldTransaction   string          `json:"last_gold_transaction"`    // nota dell'ultima transazione mostrata al GM
	// Mercato
	Market                *MarketState    `json:"market,omitempty"`
	// Tempo in-game
	GameTime   GameTime `json:"game_time"`
	HoursAwake float64  `json:"hours_awake"` // ore senza dormire
}

// FullState è il caricamento completo dello stato per un turno di gioco.
type FullState struct {
	Character  *Character
	Inventory  *Inventory
	Session    *GameSession
	SpecChoice *SpecChoiceAvailable
}

// GMCustomSkill è un'abilità unica concessa narrativamente dal GM.
type GMCustomSkill struct {
	ID          string `json:"id"`                    // slug unico, es. "tecnica_maestro_aldren"
	Name        string `json:"name"`                  // "Tecnica del Maestro Aldren"
	Description string `json:"description"`           // descrizione narrativa
	Type        string `json:"type"`                  // "active" | "passive"
	EffectDesc  string `json:"effect_desc"`           // come il GM la risolve
	MPCost      int    `json:"mp_cost,omitempty"`
	STMCost     int    `json:"stm_cost,omitempty"`
	Cooldown    int    `json:"cooldown,omitempty"`
	Origin      string `json:"origin"`                // "Chi/cosa l'ha insegnata" — contesto narrativo
	Level       int    `json:"level"`                 // livello corrente della skill (0 = base/non potenziata)
	MaxLevel    int    `json:"max_level"`             // livello massimo (0 = usa default 5)
}

// ContentGenRequest è una richiesta al Content Generator Agent di creare un nuovo documento nella KB.
type ContentGenRequest struct {
	Type    string `json:"type"`    // npc | zone | dungeon | lore | quest_context
	Subject string `json:"subject"` // nome dell'entità da generare, es. "Xander Blackthorn"
	Context string `json:"context"` // breve descrizione da espandere in un documento completo
}

// GMResponse è la risposta strutturata del GM AI.
type GMResponse struct {
	Narrative      string               `json:"narrative"`
	ContextMemo    string               `json:"context_memo,omitempty"`
	StateUpdates   *StateUpdate         `json:"state_updates,omitempty"`
	BattleTags     []string             `json:"battle_tags,omitempty"`
	UIEvents       []string             `json:"ui_events,omitempty"`
	WorldFlags     []GMWorldFlag        `json:"world_flags,omitempty"`
	CustomSkills   []GMCustomSkill      `json:"custom_skills,omitempty"`
	ActionCategory string               `json:"action_category,omitempty"`
	ContentGen     []ContentGenRequest  `json:"content_gen,omitempty"`
}

// StateUpdate contiene gli aggiornamenti di stato dal GM.
type StateUpdate struct {
	Player    *PlayerUpdate      `json:"player,omitempty"`
	GameState *GameStateUpdate   `json:"game_state,omitempty"`
	Quests    *QuestsStateUpdate `json:"quests,omitempty"`
}

// PlayerUpdate contiene gli aggiornamenti al profilo del personaggio.
type PlayerUpdate struct {
	Stats         *StatsUpdate   `json:"stats,omitempty"`
	StatusEffects []StatusEffect `json:"status_effects,omitempty"`
	Reputation    *Reputation    `json:"reputation,omitempty"`
}

// StatsUpdate contiene aggiornamenti parziali alle statistiche.
type StatsUpdate struct {
	HP  *ResourceUpdate `json:"HP,omitempty"`
	MP  *ResourceUpdate `json:"MP,omitempty"`
	STM *ResourceUpdate `json:"STM,omitempty"`
}

// ResourceUpdate aggiorna current e/o max di una risorsa.
type ResourceUpdate struct {
	Current *int `json:"current,omitempty"`
	Max     *int `json:"max,omitempty"`
}

// GameStateUpdate contiene aggiornamenti al game_session.
type GameStateUpdate struct {
	Location     string  `json:"location,omitempty"`
	SubLocation  string  `json:"sub_location,omitempty"`
	ZoneType     string  `json:"zone_type,omitempty"`
	CombatActive *bool   `json:"combat_active,omitempty"`
	CurrentEnemy *Enemy  `json:"current_enemy,omitempty"`
}

// SSEEvent è un evento inviato al client via Server-Sent Events.
type SSEEvent struct {
	Type    string `json:"type"`              // "token" | "done" | "error"
	Text    string `json:"text,omitempty"`    // narrative chunk (type=token)
	Payload any    `json:"payload,omitempty"` // stato completo (type=done)
}

// LootResult è il risultato del loot drop generato server-side.
type LootResult struct {
	Gold  int    `json:"gold"`
	Items []Item `json:"items"`
}

// DonePayload è il payload dell'evento SSE finale.
type DonePayload struct {
	Narrative  string        `json:"narrative"`
	UIEvents   []string      `json:"ui_events"`
	Character  *Character    `json:"character"`
	Inventory  *Inventory    `json:"inventory"`
	Session    *GameSession  `json:"session"`
	LevelUp    bool          `json:"level_up,omitempty"`
	NewLevel   int           `json:"new_level,omitempty"`
	Loot       *LootResult   `json:"loot,omitempty"`
	Overdrive  bool          `json:"overdrive,omitempty"`
	PlayerDied bool          `json:"player_died,omitempty"`
	GameTime   GameTime      `json:"game_time"`
}
