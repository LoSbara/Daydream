package api

import (
	"github.com/gin-gonic/gin"
	"daydream/internal/auth"
)

// GET /api/catalog/classes — catalogo classi (pubblico, no character richiesto)
func (h *Handler) ListClasses(c *gin.Context) {
	qr, err := h.DB.QueryOne("SELECT * FROM class_catalog ORDER BY id", nil)
	if err != nil {
		internalError(c, err)
		return
	}
	var classes []map[string]any
	if qr.All(&classes) != nil {
		classes = []map[string]any{}
	}
	ok(c, gin.H{"classes": classes})
}

// GET /api/catalog/monsters — catalogo mostri (pubblico)
func (h *Handler) ListMonsters(c *gin.Context) {
	qr, err := h.DB.QueryOne("SELECT * FROM monster_catalog", nil)
	if err != nil {
		internalError(c, err)
		return
	}
	var monsters []map[string]any
	if qr.All(&monsters) != nil {
		monsters = []map[string]any{}
	}
	ok(c, gin.H{"monsters": monsters})
}

// GET /api/catalog/diary — diario di viaggio del personaggio corrente
func (h *Handler) ListDiary(c *gin.Context) {
	userID := auth.GetUserID(c)
	char, _, err := h.loadSessionForUser(userID)
	if err != nil {
		handleSessionError(c, err)
		return
	}

	qr, err := h.DB.QueryOne(
		"SELECT * FROM travel_diary WHERE character_id = $cid ORDER BY created_at DESC LIMIT 50",
		map[string]any{"cid": char.ID},
	)
	if err != nil {
		internalError(c, err)
		return
	}
	var entries []map[string]any
	if qr.All(&entries) != nil {
		entries = []map[string]any{}
	}
	ok(c, gin.H{"entries": entries})
}

// GET /api/catalog/bestiary — bestiario del personaggio corrente
func (h *Handler) ListBestiary(c *gin.Context) {
	userID := auth.GetUserID(c)
	char, _, err := h.loadSessionForUser(userID)
	if err != nil {
		handleSessionError(c, err)
		return
	}

	qr, err := h.DB.QueryOne(
		"SELECT * FROM bestiary_entry WHERE character_id = $cid ORDER BY first_encountered DESC LIMIT 100",
		map[string]any{"cid": char.ID},
	)
	if err != nil {
		internalError(c, err)
		return
	}
	var entries []map[string]any
	if qr.All(&entries) != nil {
		entries = []map[string]any{}
	}
	ok(c, gin.H{"entries": entries})
}

// GET /api/catalog/npcs — NPC conosciuti dal personaggio
func (h *Handler) ListNPCs(c *gin.Context) {
	userID := auth.GetUserID(c)
	char, _, err := h.loadSessionForUser(userID)
	if err != nil {
		handleSessionError(c, err)
		return
	}

	qr, err := h.DB.QueryOne(
		"SELECT * FROM npc_instance WHERE character_id = $cid ORDER BY last_seen DESC LIMIT 50",
		map[string]any{"cid": char.ID},
	)
	if err != nil {
		internalError(c, err)
		return
	}
	var npcs []map[string]any
	if qr.All(&npcs) != nil {
		npcs = []map[string]any{}
	}
	ok(c, gin.H{"npcs": npcs})
}

