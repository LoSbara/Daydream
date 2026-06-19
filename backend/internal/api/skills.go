package api

import (
	"errors"
	"net/http"
	"daydream/internal/auth"
	"daydream/internal/db"
	"daydream/internal/models"

	"github.com/gin-gonic/gin"
)

// SkillView è la rappresentazione della skill arricchita con lo stato del personaggio.
type SkillView struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	Job              string         `json:"job"`
	Tier             int            `json:"tier"`
	MPCost           int            `json:"mp_cost"`
	STMCost          int            `json:"stm_cost"`
	CooldownTurns    int            `json:"cooldown_turns"`
	Tags             []string       `json:"tags"`
	DamageMultiplier float64        `json:"damage_multiplier,omitempty"`
	HealAmount       int            `json:"heal_amount,omitempty"`
	Element          string         `json:"element,omitempty"`
	UnlockCondition  map[string]any `json:"unlock_condition,omitempty"`
	IsUnlocked       bool           `json:"is_unlocked"`
	CooldownRemaining int           `json:"cooldown_remaining"`
	IsInLoadout      bool           `json:"is_in_loadout"`
}

// GET /api/skills — restituisce le skill disponibili per la classe del personaggio.
func (h *Handler) GetSkills(c *gin.Context) {
	userID := auth.GetUserID(c)

	char, sess, err := h.loadCharAndSession(userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "nessun personaggio trovato"})
			return
		}
		internalError(c, err)
		return
	}

	classSkills := h.Skills.ForJob(char.Job)
	loadoutSet := make(map[string]bool, len(sess.SkillLoadout))
	for _, sid := range sess.SkillLoadout {
		loadoutSet[sid] = true
	}

	views := make([]SkillView, 0, len(classSkills))
	for _, s := range classSkills {
		cooldown := 0
		if v, ok := char.SkillCooldowns[s.ID]; ok {
			cooldown = v
		}
		views = append(views, SkillView{
			ID:                s.ID,
			Name:              s.Name,
			Description:       s.Description,
			Job:               s.Job,
			Tier:              s.Tier,
			MPCost:            s.MPCost,
			STMCost:           s.STMCost,
			CooldownTurns:     s.CooldownTurns,
			Tags:              s.Tags,
			DamageMultiplier:  s.DamageMultiplier,
			HealAmount:        s.HealAmount,
			Element:           s.Element,
			UnlockCondition:   s.UnlockCondition,
			IsUnlocked:        h.Skills.IsUnlocked(s, char),
			CooldownRemaining: cooldown,
			IsInLoadout:       loadoutSet[s.ID],
		})
	}

	ok(c, gin.H{"skills": views, "loadout": sess.SkillLoadout, "skill_slots": char.SkillSlots})
}

// PUT /api/character/loadout — aggiorna il loadout di skill attive.
//
// Body: {"loadout": ["skill_id_1", "skill_id_2"]}
// Massimo char.SkillSlots skill nel loadout.
func (h *Handler) UpdateLoadout(c *gin.Context) {
	userID := auth.GetUserID(c)

	var body struct {
		Loadout []string `json:"loadout" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "loadout obbligatorio (array di skill id)")
		return
	}

	char, sess, err := h.loadCharAndSession(userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "nessun personaggio trovato"})
			return
		}
		internalError(c, err)
		return
	}

	if len(body.Loadout) > char.SkillSlots {
		badRequest(c, "troppi slot nel loadout")
		return
	}

	// Valida che tutte le skill esistano e appartengano alla classe
	for _, sid := range body.Loadout {
		s := h.Skills.Get(sid)
		if s == nil {
			badRequest(c, "skill non trovata: "+sid)
			return
		}
		if s.Job != char.Job {
			badRequest(c, "skill non disponibile per questa classe: "+sid)
			return
		}
	}

	sess.SkillLoadout = body.Loadout
	if err := h.DB.UpdateRecord(sess.ID, sess); err != nil {
		internalError(c, err)
		return
	}

	ok(c, gin.H{"loadout": sess.SkillLoadout})
}

// loadCharAndSession carica Character e GameSession di un utente.
func (h *Handler) loadCharAndSession(userID string) (*models.Character, *models.GameSession, error) {
	charQR, err := h.DB.QueryOne(
		"SELECT * FROM character WHERE user_id = $uid",
		map[string]any{"uid": userID},
	)
	if err != nil {
		return nil, nil, err
	}
	var char models.Character
	if err := charQR.First(&char); err != nil {
		return nil, nil, err
	}

	sessQR, err := h.DB.QueryOne(
		"SELECT * FROM game_session WHERE character_id = $cid",
		map[string]any{"cid": char.ID},
	)
	if err != nil {
		return nil, nil, err
	}
	var sess models.GameSession
	if err := sessQR.First(&sess); err != nil {
		return nil, nil, err
	}

	return &char, &sess, nil
}
