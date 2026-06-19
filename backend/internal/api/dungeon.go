package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"daydream/internal/auth"
	"daydream/internal/db"
	"daydream/internal/game"
	"daydream/internal/models"
)

// GET /api/dungeon — lista dungeon disponibili
func (h *Handler) ListDungeons(c *gin.Context) {
	ok(c, game.AvailableDungeons())
}

// POST /api/dungeon/enter — entra in un dungeon
// Body: { "dungeon_id": "caverna_oscura", "difficulty": 2 }
func (h *Handler) EnterDungeon(c *gin.Context) {
	userID := auth.GetUserID(c)

	var body struct {
		DungeonID  string `json:"dungeon_id" binding:"required"`
		Difficulty int    `json:"difficulty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "dungeon_id obbligatorio")
		return
	}
	if body.Difficulty < 1 || body.Difficulty > 5 {
		body.Difficulty = 1
	}

	state, sess, err := h.loadSessionForUser(userID)
	if err != nil {
		handleSessionError(c, err)
		return
	}

	// Non si può entrare in un dungeon durante un combattimento o già in un dungeon
	if sess.CombatActive {
		c.JSON(http.StatusConflict, gin.H{"error": "non puoi entrare in un dungeon durante un combattimento"})
		return
	}
	if sess.ActiveDungeon != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "sei già in un dungeon, esci prima"})
		return
	}

	dungeon, err := game.GenerateDungeon(body.DungeonID, state.Level, body.Difficulty)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dungeon.EnteredAt = sess.TurnID

	sess.ActiveDungeon = dungeon
	sess.GameState = models.StateDungeonExplore
	sess.ZoneType = "dungeon"
	sess.Location = dungeon.Name
	sess.SubLocation = game.CurrentDungeonRoom(dungeon).Name

	if err := h.DB.UpdateRecord(sess.ID, sess); err != nil {
		internalError(c, err)
		return
	}

	ok(c, gin.H{
		"dungeon":      dungeon,
		"current_room": game.CurrentDungeonRoom(dungeon),
		"session":      sess,
	})
}

// GET /api/dungeon/map — mappa scoperta del dungeon corrente
func (h *Handler) DungeonMap(c *gin.Context) {
	userID := auth.GetUserID(c)

	_, sess, err := h.loadSessionForUser(userID)
	if err != nil {
		handleSessionError(c, err)
		return
	}

	if sess.ActiveDungeon == nil {
		c.JSON(http.StatusPreconditionRequired, gin.H{"error": "non sei in un dungeon"})
		return
	}

	ok(c, gin.H{
		"dungeon":          sess.ActiveDungeon,
		"discovered_rooms": game.DiscoveredRooms(sess.ActiveDungeon),
		"current_room":     sess.ActiveDungeon.CurrentRoom,
	})
}

// POST /api/dungeon/exit — esci dal dungeon (solo dall'ingresso)
func (h *Handler) ExitDungeon(c *gin.Context) {
	userID := auth.GetUserID(c)

	_, sess, err := h.loadSessionForUser(userID)
	if err != nil {
		handleSessionError(c, err)
		return
	}

	if sess.ActiveDungeon == nil {
		c.JSON(http.StatusPreconditionRequired, gin.H{"error": "non sei in un dungeon"})
		return
	}
	if sess.CombatActive {
		c.JSON(http.StatusConflict, gin.H{"error": "non puoi uscire durante un combattimento"})
		return
	}

	// Puoi uscire solo dall'ingresso (IsEntrance) o se sei ferito gravemente
	currRoom := game.CurrentDungeonRoom(sess.ActiveDungeon)
	if currRoom != nil && !currRoom.IsEntrance {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "puoi uscire solo dall'ingresso del dungeon",
			"hint":  "torna alla stanza d'ingresso per uscire",
		})
		return
	}

	sess.ActiveDungeon = nil
	sess.GameState = models.StateWorldNavigation
	sess.ZoneType = "combat_zone"
	sess.Location = "Nexus — Uscita Dungeon"
	sess.SubLocation = ""

	if err := h.DB.UpdateRecord(sess.ID, sess); err != nil {
		internalError(c, err)
		return
	}

	ok(c, gin.H{"message": "sei uscito dal dungeon", "session": sess})
}

// ── helpers ──────────────────────────────────────────────────────────────────

func (h *Handler) loadSessionForUser(userID string) (*models.Character, *models.GameSession, error) {
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

func handleSessionError(c *gin.Context, err error) {
	if errors.Is(err, db.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "personaggio o sessione non trovati"})
		return
	}
	internalError(c, err)
}
