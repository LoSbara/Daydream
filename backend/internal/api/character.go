package api

import (
	"errors"
	"net/http"
	"daydream/internal/auth"
	"daydream/internal/db"
	"daydream/internal/models"

	"github.com/gin-gonic/gin"
)

// classBonuses definisce i bonus stat per ogni classe base (schema legacy, usato per retrocompatibilità).
// Base: tutti a 10. +5 stat primaria, +3 secondaria.
var classBonuses = map[string]map[string]int{
	"Mercenario": {"STR": 15, "DEX": 10, "AGI": 10, "TEC": 10, "VIT": 13, "LUC": 10},
	"Scout":      {"STR": 10, "DEX": 15, "AGI": 13, "TEC": 10, "VIT": 10, "LUC": 10},
	"Mago":       {"STR": 10, "DEX": 10, "AGI": 10, "TEC": 15, "VIT": 10, "LUC": 13},
	"Sacerdote":  {"STR": 10, "DEX": 10, "AGI": 10, "TEC": 10, "VIT": 15, "LUC": 13},
	"Ingegnere":  {"STR": 13, "DEX": 10, "AGI": 10, "TEC": 15, "VIT": 10, "LUC": 10},
}

// classBonusDeltas: bonus narrativi di classe (+3 a 2 stat).
// Usati nel nuovo sistema con pool di stat.
var classBonusDeltas = map[string]map[string]int{
	"Mercenario": {"STR": 3, "VIT": 3},
	"Scout":      {"DEX": 3, "AGI": 3},
	"Mago":       {"TEC": 3, "LUC": 3},
	"Sacerdote":  {"VIT": 3, "TEC": 3},
	"Ingegnere":  {"TEC": 3, "DEX": 3},
}

// validJobs è l'insieme delle classi selezionabili.
var validJobs = map[string]bool{
	"Mercenario": true, "Scout": true, "Mago": true, "Sacerdote": true, "Ingegnere": true,
}

// POST /api/character — crea il personaggio per l'utente autenticato.
// Un utente può avere un solo personaggio (Phase 1).
func (h *Handler) CreateCharacter(c *gin.Context) {
	userID := auth.GetUserID(c)

	var body struct {
		Name         string         `json:"name"          binding:"required,min=2,max=24"`
		Job          string         `json:"job"           binding:"required"`
		InitialStats map[string]int `json:"initial_stats"` // opzionale: pool 15 punti, base 5 per stat
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "name (2-24 char) e job obbligatori")
		return
	}
	if !validJobs[body.Job] {
		badRequest(c, "job non valido: scegli tra Mercenario, Scout, Mago, Sacerdote, Ingegnere")
		return
	}

	// Valida initial_stats se forniti
	if body.InitialStats != nil {
		validAttrs := map[string]bool{"STR": true, "DEX": true, "AGI": true, "TEC": true, "VIT": true, "LUC": true}
		sum := 0
		for attr, val := range body.InitialStats {
			if !validAttrs[attr] {
				badRequest(c, "stat non valida: "+attr)
				return
			}
			if val < 5 {
				badRequest(c, "ogni stat deve essere almeno 5")
				return
			}
			sum += val
		}
		// sum totale <= 45 (30 base + 15 pool): base 5×6=30, pool 15
		if sum-30 > 15 {
			badRequest(c, "punti stat totali superano il massimo consentito (15 punti extra su base 5)")
			return
		}
	}

	// Controlla se esiste già un personaggio
	results, err := h.DB.Query(
		"SELECT id FROM character WHERE user_id = $uid",
		map[string]any{"uid": userID},
	)
	if err != nil {
		internalError(c, err)
		return
	}
	var existing []struct{ ID string `json:"id"` }
	if results[0].All(&existing) == nil && len(existing) > 0 {
		conflict(c, "hai già un personaggio attivo")
		return
	}

	// Calcola stats iniziali: usa initial_stats se forniti, altrimenti default legacy
	var stats models.Stats
	if body.InitialStats != nil {
		stats = buildInitialStatsFromPool(body.Job, body.InitialStats)
	} else {
		stats = buildInitialStats(body.Job)
	}

	// Crea character
	char := models.Character{
		UserID:           userID,
		Name:             body.Name,
		Job:              body.Job,
		Level:            1,
		Experience:       0,
		ExperienceToNext: 100,
		Stats:            stats,
		Money:            500,
		SkillSlots:       4,
		Titles:           []string{},
		StatusEffects:    []models.StatusEffect{},
		Reputation:       models.Reputation{},
		Flags:            map[string]any{},
		ActionCounters:   models.ActionCounters{ZonesVisited: []string{"Nexus"}},
		SkillCooldowns:   map[string]int{},
	}

	var createdChar models.Character
	if err := h.DB.CreateRecord("character", char, &createdChar); err != nil {
		internalError(c, err)
		return
	}

	// Crea inventory (vuoto)
	inv := models.Inventory{
		CharacterID:                createdChar.ID,
		Equipped:                   models.EquippedSlots{},
		StatBonusesFromEquipment:   models.StatBonuses{},
		Bag:                        []models.Item{},
	}
	var createdInv models.Inventory
	if err := h.DB.CreateRecord("inventory", inv, &createdInv); err != nil {
		internalError(c, err)
		return
	}

	// Crea game_session
	sess := models.GameSession{
		CharacterID:            createdChar.ID,
		Location:               "Nexus",
		SubLocation:            "",
		ZoneType:               "safe_zone",
		GameState:              models.StateWorldNavigation,
		CombatActive:           false,
		SkillLoadout:           []string{},
		SessionLog:             []models.SessionMessage{},
		ContextMemo:            "",
		PendingNarrativeEvents: []string{"INTRO: Questa è la prima sessione del giocatore. Narra il loro arrivo a Nexus e presentati come GM. Chiedi se vogliono esplorare la città o uscire in avventura."},
		QuestsActive:           []models.Quest{},
		QuestsCompleted:        []models.Quest{},
		TurnID:                 0,
	}
	var createdSess models.GameSession
	if err := h.DB.CreateRecord("game_session", sess, &createdSess); err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"character": createdChar,
			"inventory": createdInv,
			"session":   createdSess,
		},
	})
}

// GET /api/character — restituisce il personaggio dell'utente autenticato.
func (h *Handler) GetCharacter(c *gin.Context) {
	userID := auth.GetUserID(c)

	results, err := h.DB.Query(
		"SELECT * FROM character WHERE user_id = $uid",
		map[string]any{"uid": userID},
	)
	if err != nil {
		internalError(c, err)
		return
	}

	var char models.Character
	if err := results[0].First(&char); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "nessun personaggio trovato"})
			return
		}
		internalError(c, err)
		return
	}

	ok(c, char)
}

// PUT /api/character/stats — spende un punto stat su un attributo.
func (h *Handler) AllocateStats(c *gin.Context) {
	userID := auth.GetUserID(c)

	var body struct {
		Attribute string `json:"attribute" binding:"required"`
		Points    int    `json:"points"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "attribute obbligatorio")
		return
	}
	if body.Points <= 0 {
		body.Points = 1
	}

	validAttrs := map[string]bool{"STR": true, "DEX": true, "AGI": true, "TEC": true, "VIT": true, "LUC": true}
	if !validAttrs[body.Attribute] {
		badRequest(c, "attribute non valido: scegli tra STR, DEX, AGI, TEC, VIT, LUC")
		return
	}

	results, err := h.DB.Query(
		"SELECT * FROM character WHERE user_id = $uid",
		map[string]any{"uid": userID},
	)
	if err != nil {
		internalError(c, err)
		return
	}
	var char models.Character
	if err := results[0].First(&char); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "personaggio non trovato"})
		return
	}

	if char.StatPointsAvailable < body.Points {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "punti stat insufficienti"})
		return
	}

	// Applica i punti
	char.StatPointsAvailable -= body.Points
	for i := 0; i < body.Points; i++ {
		switch body.Attribute {
		case "STR":
			char.Stats.STR++
		case "DEX":
			char.Stats.DEX++
		case "AGI":
			char.Stats.AGI++
		case "TEC":
			char.Stats.TEC++
			char.Stats.MP.Max += 3 // +3 MP max per TEC
		case "VIT":
			char.Stats.VIT++
			char.Stats.HP.Max += 5 // +5 HP max per VIT
		case "LUC":
			char.Stats.LUC++
		}
	}

	if err := h.DB.UpdateRecord(char.ID, char); err != nil {
		internalError(c, err)
		return
	}

	ok(c, char)
}

// buildInitialStatsFromPool calcola le stat iniziali dal pool di allocazione del giocatore.
// initialStats: valori grezzi (base 5 + punti allocati, senza bonus classe).
// Il bonus classe viene aggiunto sopra i valori scelti dal giocatore.
func buildInitialStatsFromPool(job string, initialStats map[string]int) models.Stats {
	deltas := classBonusDeltas[job]

	getVal := func(attr string) int {
		v := initialStats[attr]
		if v < 5 {
			v = 5
		}
		return v + deltas[attr]
	}

	str := getVal("STR")
	dex := getVal("DEX")
	agi := getVal("AGI")
	tec := getVal("TEC")
	vit := getVal("VIT")
	luc := getVal("LUC")

	// HP = 100 + (VIT-10)*5, MP = 50 + (TEC-10)*3, STM = 100
	maxHP := 100 + (vit-10)*5
	if maxHP < 50 {
		maxHP = 50
	}
	maxMP := 50 + (tec-10)*3
	if maxMP < 20 {
		maxMP = 20
	}
	maxSTM := 100

	return models.Stats{
		HP:  models.Resource{Current: maxHP, Max: maxHP},
		MP:  models.Resource{Current: maxMP, Max: maxMP},
		STM: models.Resource{Current: maxSTM, Max: maxSTM},
		STR: str,
		DEX: dex,
		AGI: agi,
		TEC: tec,
		VIT: vit,
		LUC: luc,
	}
}

// buildInitialStats calcola le statistiche iniziali di un personaggio.
func buildInitialStats(job string) models.Stats {
	bonuses := classBonuses[job]

	str := bonuses["STR"]
	dex := bonuses["DEX"]
	agi := bonuses["AGI"]
	tec := bonuses["TEC"]
	vit := bonuses["VIT"]
	luc := bonuses["LUC"]

	// HP = 100 + (VIT-10)*5, MP = 50 + (TEC-10)*3, STM = 100
	maxHP := 100 + (vit-10)*5
	maxMP := 50 + (tec-10)*3
	maxSTM := 100

	return models.Stats{
		HP:  models.Resource{Current: maxHP, Max: maxHP},
		MP:  models.Resource{Current: maxMP, Max: maxMP},
		STM: models.Resource{Current: maxSTM, Max: maxSTM},
		STR: str,
		DEX: dex,
		AGI: agi,
		TEC: tec,
		VIT: vit,
		LUC: luc,
	}
}
