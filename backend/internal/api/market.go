package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"daydream/internal/auth"
	"daydream/internal/game"
	"daydream/internal/models"
)

// GET /api/market/browse — genera o restituisce il mercato corrente
func (h *Handler) BrowseMarket(c *gin.Context) {
	userID := auth.GetUserID(c)

	char, sess, err := h.loadCharAndSession(userID)
	if err != nil {
		handleSessionError(c, err)
		return
	}

	// Se location è cambiata o mercato non esiste, rigenera
	if sess.Market == nil || sess.Market.Location != sess.Location {
		sess.Market = game.GenerateMarketListings(
			char.Stats.LUC, char.Stats.TEC, char.Level, sess.Location,
		)
		if err := h.DB.UpdateRecord(sess.ID, sess); err != nil {
			internalError(c, err)
			return
		}
	}

	ok(c, sess.Market)
}

// POST /api/market/buy — acquista un item dal mercato
func (h *Handler) BuyMarketItem(c *gin.Context) {
	userID := auth.GetUserID(c)

	var body struct {
		ItemID string `json:"item_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	char, inv, sess, err := h.loadCharInvSession(userID)
	if err != nil {
		handleSessionError(c, err)
		return
	}

	if sess.Market == nil {
		badRequest(c, "nessun mercato attivo")
		return
	}

	// Trova il listing
	listingIdx := -1
	for i, l := range sess.Market.Listings {
		if l.Item.ID == body.ItemID && !l.Sold {
			listingIdx = i
			break
		}
	}
	if listingIdx < 0 {
		badRequest(c, "item non trovato o già venduto")
		return
	}

	listing := sess.Market.Listings[listingIdx]

	// Controlla oro
	if char.Money < listing.Price {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "oro insufficiente"})
		return
	}

	// Deduce oro, aggiunge item alla borsa
	char.Money -= listing.Price
	inv.Bag = append(inv.Bag, listing.Item)
	sess.Market.Listings[listingIdx].Sold = true

	// Salva tutto
	if err := h.DB.UpdateRecord(char.ID, char); err != nil {
		internalError(c, err)
		return
	}
	if err := h.DB.UpdateRecord(inv.ID, inv); err != nil {
		internalError(c, err)
		return
	}
	if err := h.DB.UpdateRecord(sess.ID, sess); err != nil {
		internalError(c, err)
		return
	}

	ok(c, gin.H{"character": char, "inventory": inv, "market": sess.Market})
}

// POST /api/market/negotiate — contratta il prezzo di un item
func (h *Handler) NegotiateMarketItem(c *gin.Context) {
	userID := auth.GetUserID(c)

	var body struct {
		ItemID string `json:"item_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	char, sess, err := h.loadCharAndSession(userID)
	if err != nil {
		handleSessionError(c, err)
		return
	}

	if sess.Market == nil {
		badRequest(c, "nessun mercato attivo")
		return
	}

	listingIdx := -1
	for i, l := range sess.Market.Listings {
		if l.Item.ID == body.ItemID && !l.Sold && !l.NegoDone {
			listingIdx = i
			break
		}
	}
	if listingIdx < 0 {
		badRequest(c, "item non trovato, già venduto, o contrattazione già effettuata")
		return
	}

	listing := sess.Market.Listings[listingIdx]
	result := game.NegotiatePrice(listing, char.Stats.LUC, char.Stats.TEC)

	// Aggiorna prezzo se successo
	sess.Market.Listings[listingIdx].Price = result.NewPrice
	sess.Market.Listings[listingIdx].NegoDone = true

	if err := h.DB.UpdateRecord(sess.ID, sess); err != nil {
		internalError(c, err)
		return
	}

	ok(c, result)
}

// POST /api/market/analyze — analizza un item del mercato prima di comprarlo
func (h *Handler) AnalyzeMarketItem(c *gin.Context) {
	userID := auth.GetUserID(c)
	var body struct {
		ItemID string `json:"item_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	char, _, sess, err := h.loadCharInvSession(userID)
	if err != nil {
		handleSessionError(c, err)
		return
	}
	if sess.Market == nil {
		badRequest(c, "nessun mercato attivo")
		return
	}

	listingIdx := -1
	for i, l := range sess.Market.Listings {
		if l.Item.ID == body.ItemID && !l.Sold {
			listingIdx = i
			break
		}
	}
	if listingIdx < 0 {
		badRequest(c, "item non trovato")
		return
	}
	if sess.Market.Listings[listingIdx].Analyzed {
		// già analizzato: ritorna il mercato senza ricalcolare
		ok(c, sess.Market)
		return
	}

	// Calcola qualità dell'analisi in base a TEC
	item := sess.Market.Listings[listingIdx].Item
	analysis := game.ApplyTecAnalysis(item, char.Stats.TEC, char.Stats.LUC)
	sess.Market.Listings[listingIdx].Item.PerceivedStatBonus = analysis.Stats
	sess.Market.Listings[listingIdx].Item.PerceivedRarity = analysis.Rarity
	sess.Market.Listings[listingIdx].Analyzed = true

	if err := saveSession(h, sess); err != nil {
		internalError(c, err)
		return
	}
	ok(c, sess.Market)
}

// ── helpers ──────────────────────────────────────────────────────────────────

// saveSession persiste la GameSession nel DB.
func saveSession(h *Handler, sess *models.GameSession) error {
	return h.DB.UpdateRecord(sess.ID, sess)
}

// loadCharInvSession carica Character, Inventory e GameSession dato uno userID.
func (h *Handler) loadCharInvSession(userID string) (*models.Character, *models.Inventory, *models.GameSession, error) {
	char, sess, err := h.loadCharAndSession(userID)
	if err != nil {
		return nil, nil, nil, err
	}

	invQR, err := h.DB.QueryOne(
		"SELECT * FROM inventory WHERE character_id = $cid",
		map[string]any{"cid": char.ID},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	var inv models.Inventory
	if err := invQR.First(&inv); err != nil {
		return nil, nil, nil, err
	}

	return char, &inv, sess, nil
}
