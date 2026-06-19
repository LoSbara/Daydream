package api

import (
	"net/http"

	"daydream/internal/auth"
	"daydream/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetWorldFlags(c *gin.Context) {
	userID := auth.GetUserID(c)

	worldQR, err := h.DB.QueryOne(
		"SELECT * FROM character WHERE user_id = $uid LIMIT 1",
		map[string]any{"uid": userID},
	)
	if err != nil {
		internalError(c, err)
		return
	}
	var chars []models.Character
	if err := worldQR.All(&chars); err != nil || len(chars) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "personaggio non trovato"})
		return
	}

	allResults, err := h.DB.Query(
		"SELECT * FROM world_flags WHERE character_id = $cid ORDER BY updated_at DESC",
		map[string]any{"cid": chars[0].ID},
	)
	if err != nil {
		ok(c, []models.WorldFlag{})
		return
	}
	var flags []models.WorldFlag
	if err := allResults[0].All(&flags); err != nil {
		ok(c, []models.WorldFlag{})
		return
	}
	ok(c, flags)
}
