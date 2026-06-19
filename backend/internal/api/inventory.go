package api

import (
	"errors"
	"net/http"
	"daydream/internal/auth"
	"daydream/internal/db"
	"daydream/internal/game"
	"daydream/internal/models"

	"github.com/gin-gonic/gin"
)

var validSlots = map[string]bool{
	"weapon":      true,
	"offhand":     true,
	"head":        true,
	"chest":       true,
	"legs":        true,
	"boots":       true,
	"accessory_1": true,
	"accessory_2": true,
}

// POST /api/inventory/equip — equipaggia un item dalla borsa in uno slot.
//
// Body: {"item_id": "xxx", "slot": "weapon"}
func (h *Handler) EquipItem(c *gin.Context) {
	userID := auth.GetUserID(c)

	var body struct {
		ItemID string `json:"item_id" binding:"required"`
		Slot   string `json:"slot"    binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "item_id e slot obbligatori")
		return
	}

	if !validSlots[body.Slot] {
		badRequest(c, "slot non valido: "+body.Slot)
		return
	}

	_, inv, err := h.loadCharAndInventory(userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "nessun personaggio trovato"})
			return
		}
		internalError(c, err)
		return
	}

	// Trova item nella borsa
	itemIdx := -1
	for i, item := range inv.Bag {
		if item.ID == body.ItemID {
			itemIdx = i
			break
		}
	}
	if itemIdx < 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "item non trovato nella borsa"})
		return
	}

	newItem := inv.Bag[itemIdx]

	// Se c'è già qualcosa nello slot, rimettilo in borsa
	if old := slotItem(&inv.Equipped, body.Slot); old != nil {
		inv.Bag = append(inv.Bag, *old)
	}

	// Equipaggia il nuovo item e rimuovilo dalla borsa
	setSlotItem(&inv.Equipped, body.Slot, &newItem)
	inv.Bag = append(inv.Bag[:itemIdx], inv.Bag[itemIdx+1:]...)

	inv.StatBonusesFromEquipment = recalcStatBonuses(inv.Equipped)

	if err := h.DB.UpdateRecord(inv.ID, inv); err != nil {
		internalError(c, err)
		return
	}

	ok(c, gin.H{"inventory": inv})
}

// POST /api/inventory/unequip — rimuove un item da uno slot e lo rimette in borsa.
//
// Body: {"slot": "weapon"}
func (h *Handler) UnequipItem(c *gin.Context) {
	userID := auth.GetUserID(c)

	var body struct {
		Slot string `json:"slot" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "slot obbligatorio")
		return
	}

	if !validSlots[body.Slot] {
		badRequest(c, "slot non valido: "+body.Slot)
		return
	}

	_, inv, err := h.loadCharAndInventory(userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "nessun personaggio trovato"})
			return
		}
		internalError(c, err)
		return
	}

	item := slotItem(&inv.Equipped, body.Slot)
	if item == nil {
		badRequest(c, "slot già vuoto")
		return
	}

	inv.Bag = append(inv.Bag, *item)
	setSlotItem(&inv.Equipped, body.Slot, nil)

	inv.StatBonusesFromEquipment = recalcStatBonuses(inv.Equipped)

	if err := h.DB.UpdateRecord(inv.ID, inv); err != nil {
		internalError(c, err)
		return
	}

	ok(c, gin.H{"inventory": inv})
}

// loadCharAndInventory carica Character e Inventory di un utente.
func (h *Handler) loadCharAndInventory(userID string) (*models.Character, *models.Inventory, error) {
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

	invQR, err := h.DB.QueryOne(
		"SELECT * FROM inventory WHERE character_id = $cid",
		map[string]any{"cid": char.ID},
	)
	if err != nil {
		return nil, nil, err
	}
	var inv models.Inventory
	if err := invQR.First(&inv); err != nil {
		return nil, nil, err
	}

	return &char, &inv, nil
}

// slotItem legge l'item da uno slot dato il nome.
func slotItem(e *models.EquippedSlots, slot string) *models.Item {
	switch slot {
	case "weapon":
		return e.Weapon
	case "offhand":
		return e.Offhand
	case "head":
		return e.Head
	case "chest":
		return e.Chest
	case "legs":
		return e.Legs
	case "boots":
		return e.Boots
	case "accessory_1":
		return e.Accessory1
	case "accessory_2":
		return e.Accessory2
	}
	return nil
}

// setSlotItem scrive (o cancella con nil) un item in uno slot.
func setSlotItem(e *models.EquippedSlots, slot string, item *models.Item) {
	switch slot {
	case "weapon":
		e.Weapon = item
	case "offhand":
		e.Offhand = item
	case "head":
		e.Head = item
	case "chest":
		e.Chest = item
	case "legs":
		e.Legs = item
	case "boots":
		e.Boots = item
	case "accessory_1":
		e.Accessory1 = item
	case "accessory_2":
		e.Accessory2 = item
	}
}

// recalcStatBonuses ricalcola i bonus totali da tutto l'equipaggiamento corrente.
func recalcStatBonuses(equipped models.EquippedSlots) models.StatBonuses {
	var b models.StatBonuses
	slots := []*models.Item{
		equipped.Weapon, equipped.Offhand, equipped.Head,
		equipped.Chest, equipped.Legs, equipped.Boots,
		equipped.Accessory1, equipped.Accessory2,
	}
	for _, item := range slots {
		if item == nil {
			continue
		}
		for stat, val := range item.StatBonus {
			switch stat {
			case "STR":
				b.STR += val
			case "DEX":
				b.DEX += val
			case "AGI":
				b.AGI += val
			case "TEC":
				b.TEC += val
			case "VIT":
				b.VIT += val
			case "LUC":
				b.LUC += val
			case "HP":
				b.HPBonus += val
			case "MP":
				b.MPBonus += val
			case "STM":
				b.STMBonus += val
			}
		}
	}
	return b
}

// saveCharAndInventory salva personaggio e inventario nel DB.
func saveCharAndInventory(h *Handler, char *models.Character, inv *models.Inventory) error {
	if err := h.DB.UpdateRecord(char.ID, char); err != nil {
		return err
	}
	return h.DB.UpdateRecord(inv.ID, inv)
}

// POST /api/inventory/appraise — identifica un item non appraised
func (h *Handler) AppraiseItem(c *gin.Context) {
	userID := auth.GetUserID(c)
	var body struct {
		ItemID string `json:"item_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	char, inv, err := h.loadCharAndInventory(userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "nessun personaggio trovato"})
			return
		}
		internalError(c, err)
		return
	}

	// Trova l'item in borsa (non si identificano item equipaggiati)
	itemIdx := -1
	for i, it := range inv.Bag {
		if it.ID == body.ItemID {
			itemIdx = i
			break
		}
	}
	if itemIdx < 0 {
		badRequest(c, "item non trovato nella borsa")
		return
	}

	item := inv.Bag[itemIdx]
	if item.Appraised {
		badRequest(c, "item già identificato")
		return
	}
	if item.Rarity == "common" {
		badRequest(c, "item comune, non richiede identificazione")
		return
	}

	// Formula dinamica: required_tec = item_level × rarity_mult
	rarityTecMult := map[string]float64{
		"common":    0.5,
		"uncommon":  0.8,
		"rare":      1.3,
		"epic":      2.0,
		"legendary": 3.0,
	}
	rarityGoldMult := map[string]float64{
		"common":    0.5,
		"uncommon":  0.8,
		"rare":      1.3,
		"epic":      2.0,
		"legendary": 3.0,
	}

	itemLevel := item.EnhancementLevel
	if itemLevel == 0 {
		itemLevel = 5
	}

	tecMult := rarityTecMult[item.Rarity]
	if tecMult == 0 {
		tecMult = 1.0
	}
	requiredTec := float64(itemLevel) * tecMult

	// Costo base in oro
	baseCost := int(float64(itemLevel) * rarityGoldMult[item.Rarity] * 15)
	if baseCost < 10 {
		baseCost = 10
	}

	// LUC contribuisce al 30% come TEC effettiva aggiuntiva
	effectiveTec := float64(char.Stats.TEC) + float64(char.Stats.LUC)*0.3
	var cost int
	switch {
	case effectiveTec >= requiredTec:
		cost = baseCost / 4 // 75% sconto: spese materiali minime
	case effectiveTec >= requiredTec*0.5:
		cost = baseCost / 2 // 50% sconto
	default:
		cost = baseCost // prezzo pieno
	}

	if char.Money < cost {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error": "oro insufficiente",
			"cost":  cost,
		})
		return
	}

	// Applica analisi TEC-dipendente
	char.Money -= cost
	analysis := game.ApplyTecAnalysis(item, char.Stats.TEC, char.Stats.LUC)
	inv.Bag[itemIdx].PerceivedStatBonus = analysis.Stats
	inv.Bag[itemIdx].PerceivedRarity = analysis.Rarity
	inv.Bag[itemIdx].Appraised = true

	// Salva char e inventory
	if err := saveCharAndInventory(h, char, inv); err != nil {
		internalError(c, err)
		return
	}

	ok(c, gin.H{"inventory": inv, "character": char, "cost_paid": cost})
}
