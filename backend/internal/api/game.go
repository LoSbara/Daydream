package api

import (
	"errors"
	"io"
	"net/http"
	"daydream/internal/auth"
	"daydream/internal/db"
	"daydream/internal/game"
	"daydream/internal/models"

	"github.com/gin-gonic/gin"
)

// POST /api/chat — invia un messaggio al GM e riceve la risposta via SSE.
//
// Il client deve aprire questa connessione con Accept: text/event-stream.
// Tipi di eventi SSE:
//
//	event: message  data: {"type":"token","text":"..."}   → narrative chunk in streaming
//	event: message  data: {"type":"done","payload":{...}} → stato aggiornato completo
//	event: message  data: {"type":"error","text":"..."}   → errore fatale del turno
func (h *Handler) Chat(c *gin.Context) {
	userID := auth.GetUserID(c)

	var body struct {
		Message string `json:"message" binding:"required,min=1,max=2000"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "message obbligatorio (max 2000 caratteri)")
		return
	}

	// Carica l'ID del personaggio
	results, err := h.DB.Query(
		"SELECT id FROM character WHERE user_id = $uid",
		map[string]any{"uid": userID},
	)
	if err != nil {
		internalError(c, err)
		return
	}
	var charRef struct {
		ID string `json:"id"`
	}
	if err := results[0].First(&charRef); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			c.JSON(http.StatusPreconditionRequired, gin.H{"error": "crea prima un personaggio"})
			return
		}
		internalError(c, err)
		return
	}

	// Sanifica il messaggio: rimuovi HTML, caratteri di controllo, whitespace
	msg := sanitizeMessage(body.Message)
	if msg == "" {
		badRequest(c, "messaggio vuoto dopo la sanitizzazione")
		return
	}

	// Metti il job in coda
	job, err := h.Queue.Enqueue(c.Request.Context(), userID, charRef.ID, msg)
	if err != nil {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}

	// SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // disabilita il buffering di nginx

	// Stream eventi dal TokenCh finché non viene chiuso dal worker
	c.Stream(func(_ io.Writer) bool {
		event, ok := <-job.TokenCh
		if !ok {
			return false
		}
		c.SSEvent("message", event)
		return true
	})
}

// GET /api/state — restituisce lo stato completo del personaggio.
func (h *Handler) GetState(c *gin.Context) {
	userID := auth.GetUserID(c)

	// Character
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

	// Inventory
	results, err = h.DB.Query(
		"SELECT * FROM inventory WHERE character_id = $cid",
		map[string]any{"cid": char.ID},
	)
	if err != nil {
		internalError(c, err)
		return
	}
	var inv models.Inventory
	if err := results[0].First(&inv); err != nil {
		internalError(c, err)
		return
	}

	// Session
	results, err = h.DB.Query(
		"SELECT * FROM game_session WHERE character_id = $cid",
		map[string]any{"cid": char.ID},
	)
	if err != nil {
		internalError(c, err)
		return
	}
	var sess models.GameSession
	if err := results[0].First(&sess); err != nil {
		internalError(c, err)
		return
	}

	specChoice := game.GetAvailableSpecChoice(char.Job, char.Level, char.ChosenSpecs)

	ok(c, gin.H{
		"character":   char,
		"inventory":   inv,
		"session":     sess,
		"spec_choice": specChoice,
	})
}
