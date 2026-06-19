package api

import (
	"errors"
	"net/http"
	"daydream/internal/auth"
	"daydream/internal/db"
	"daydream/internal/models"

	"github.com/gin-gonic/gin"
)

// GET /api/quests — restituisce le quest del personaggio suddivise per stato.
func (h *Handler) GetQuests(c *gin.Context) {
	userID := auth.GetUserID(c)

	_, sess, err := h.loadCharAndSession(userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "nessun personaggio trovato"})
			return
		}
		internalError(c, err)
		return
	}

	active := make([]models.Quest, 0)
	completed := make([]models.Quest, 0)
	failed := make([]models.Quest, 0)

	for _, q := range sess.QuestsActive {
		switch q.Status {
		case "active":
			active = append(active, q)
		case "completed":
			completed = append(completed, q)
		case "failed", "expired":
			failed = append(failed, q)
		}
	}
	for _, q := range sess.QuestsCompleted {
		switch q.Status {
		case "completed":
			completed = append(completed, q)
		default:
			failed = append(failed, q)
		}
	}

	ok(c, gin.H{
		"active":    active,
		"completed": completed,
		"failed":    failed,
	})
}
