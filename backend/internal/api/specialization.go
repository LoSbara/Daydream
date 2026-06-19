package api

import (
	"daydream/internal/auth"
	"daydream/internal/game"
	"daydream/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /api/character/spec-choices
func (h *Handler) GetSpecChoices(c *gin.Context) {
	userID := auth.GetUserID(c)
	chars, err := h.DB.Query("SELECT * FROM character WHERE user_id=$uid LIMIT 1", map[string]any{"uid": userID})
	if err != nil {
		internalError(c, err)
		return
	}
	var charList []models.Character
	if err := chars[0].All(&charList); err != nil || len(charList) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "personaggio non trovato"})
		return
	}
	char := charList[0]
	choice := game.GetAvailableSpecChoice(char.Job, char.Level, char.ChosenSpecs)
	if choice == nil {
		ok(c, gin.H{"spec_choice": nil})
		return
	}
	ok(c, gin.H{"spec_choice": choice})
}

// POST /api/character/spec-choice
func (h *Handler) ChooseSpec(c *gin.Context) {
	userID := auth.GetUserID(c)
	var body struct {
		SpecID string `json:"spec_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	chars, err := h.DB.Query("SELECT * FROM character WHERE user_id=$uid LIMIT 1", map[string]any{"uid": userID})
	if err != nil {
		internalError(c, err)
		return
	}
	var charList []models.Character
	if err := chars[0].All(&charList); err != nil || len(charList) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "personaggio non trovato"})
		return
	}
	char := charList[0]

	choice := game.GetAvailableSpecChoice(char.Job, char.Level, char.ChosenSpecs)
	if choice == nil {
		badRequest(c, "nessuna specializzazione disponibile")
		return
	}
	valid := false
	for _, opt := range choice.Options {
		if opt.ID == body.SpecID {
			valid = true
			break
		}
	}
	if !valid {
		badRequest(c, "specializzazione non valida")
		return
	}

	newSpecs := append(char.ChosenSpecs, body.SpecID)
	_, err = h.DB.Query(
		"UPDATE character SET chosen_specs=$specs WHERE user_id=$uid",
		map[string]any{"specs": newSpecs, "uid": userID},
	)
	if err != nil {
		internalError(c, err)
		return
	}

	char.ChosenSpecs = newSpecs
	ok(c, char)
}
