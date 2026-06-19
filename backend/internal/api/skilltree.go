package api

import (
	"fmt"
	"net/http"

	"daydream/internal/auth"
	"daydream/internal/game"
	"daydream/internal/models"

	"github.com/gin-gonic/gin"
)

// GET /api/character/skill-tree
func (h *Handler) GetSkillTree(c *gin.Context) {
	userID := auth.GetUserID(c)
	results, err := h.DB.Query("SELECT * FROM character WHERE user_id=$uid LIMIT 1", map[string]any{"uid": userID})
	if err != nil {
		internalError(c, err)
		return
	}
	var chars []models.Character
	if err := results[0].All(&chars); err != nil || len(chars) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "personaggio non trovato"})
		return
	}
	char := chars[0]
	// nil guard: SurrealDB può restituire nil per array vuoti
	if char.SkillTreeUnlocks == nil {
		char.SkillTreeUnlocks = []string{}
	}
	if char.ChosenSpecs == nil {
		char.ChosenSpecs = []string{}
	}

	// Backfill: se il personaggio ha livello > 1 ma 0 punti,
	// assegna i punti che avrebbe dovuto guadagnare (3 per livello)
	// meno quelli già spesi (len di skill_tree_unlocks)
	if char.Level > 1 && char.TreePointsAvailable == 0 {
		expectedPoints := (char.Level - 1) * 3
		spentPoints := len(char.SkillTreeUnlocks)
		if expectedPoints > spentPoints {
			char.TreePointsAvailable = expectedPoints - spentPoints
			h.DB.Query(
				"UPDATE $id SET tree_points_available=$pts",
				map[string]any{"id": char.ID, "pts": char.TreePointsAvailable},
			)
		}
	}

	tree := game.GetClassSkillTree(char.Job, char.SkillTreeUnlocks, char.ChosenSpecs)
	if tree == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "albero non trovato per questa classe"})
		return
	}
	ok(c, gin.H{"tree": tree, "points_available": char.TreePointsAvailable})
}

// POST /api/character/skill-tree/unlock
func (h *Handler) UnlockSkillTreeNode(c *gin.Context) {
	userID := auth.GetUserID(c)
	var body struct {
		SkillID string `json:"skill_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	results, err := h.DB.Query("SELECT * FROM character WHERE user_id=$uid LIMIT 1", map[string]any{"uid": userID})
	if err != nil {
		internalError(c, err)
		return
	}
	var chars []models.Character
	if err := results[0].All(&chars); err != nil || len(chars) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "personaggio non trovato"})
		return
	}
	char := chars[0]
	if char.SkillTreeUnlocks == nil {
		char.SkillTreeUnlocks = []string{}
	}
	if char.ChosenSpecs == nil {
		char.ChosenSpecs = []string{}
	}

	// Valida con GetClassSkillTree
	tree := game.GetClassSkillTree(char.Job, char.SkillTreeUnlocks, char.ChosenSpecs)
	if tree == nil {
		badRequest(c, "classe non valida")
		return
	}

	// Trova il nodo e verifica disponibilità
	var targetNode *models.SkillTreeNode
	for _, branch := range tree.Branches {
		for i, sk := range branch.Skills {
			if sk.ID == body.SkillID {
				targetNode = &branch.Skills[i]
				break
			}
		}
		if targetNode != nil {
			break
		}
	}
	if targetNode == nil {
		badRequest(c, "skill non trovata")
		return
	}
	if !targetNode.Available {
		badRequest(c, "prerequisiti non soddisfatti")
		return
	}

	cost := targetNode.Cost
	if cost == 0 {
		cost = 3 // fallback tier 1
	}
	if char.TreePointsAvailable < cost {
		badRequest(c, fmt.Sprintf("punti skill insufficienti (servono %d, hai %d)", cost, char.TreePointsAvailable))
		return
	}

	// Sblocca
	char.SkillTreeUnlocks = append(char.SkillTreeUnlocks, body.SkillID)
	char.TreePointsAvailable -= cost

	// Applica bonus stat se presenti
	for stat, bonus := range targetNode.StatBonus {
		switch stat {
		case "STR":
			char.Stats.STR += bonus
		case "DEX":
			char.Stats.DEX += bonus
		case "AGI":
			char.Stats.AGI += bonus
		case "TEC":
			char.Stats.TEC += bonus
		case "VIT":
			char.Stats.VIT += bonus
		case "LUC":
			char.Stats.LUC += bonus
		}
	}

	// Salva
	_, err = h.DB.Query(
		"UPDATE $id SET tree_points_available=$pts, skill_tree_unlocks=$unlocks, stats=$stats",
		map[string]any{
			"id":      char.ID,
			"pts":     char.TreePointsAvailable,
			"unlocks": char.SkillTreeUnlocks,
			"stats":   char.Stats,
		},
	)
	if err != nil {
		internalError(c, err)
		return
	}

	ok(c, char)
}

// POST /api/character/custom-skill/upgrade — potenzia una custom skill narrativa.
func (h *Handler) UpgradeCustomSkill(c *gin.Context) {
	userID := auth.GetUserID(c)
	var body struct {
		SkillName string `json:"skill_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err.Error())
		return
	}

	results, err := h.DB.Query("SELECT * FROM character WHERE user_id=$uid LIMIT 1", map[string]any{"uid": userID})
	if err != nil {
		internalError(c, err)
		return
	}
	var chars []models.Character
	if err := results[0].All(&chars); err != nil || len(chars) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "personaggio non trovato"})
		return
	}
	char := chars[0]

	skillIdx := -1
	for i, s := range char.CustomSkills {
		if s.Name == body.SkillName {
			skillIdx = i
			break
		}
	}
	if skillIdx < 0 {
		badRequest(c, "skill non trovata")
		return
	}

	skill := char.CustomSkills[skillIdx]
	maxLv := skill.MaxLevel
	if maxLv == 0 {
		maxLv = 5
	}
	if skill.Level >= maxLv {
		badRequest(c, "skill già al livello massimo")
		return
	}

	// Costi: 0→1 gratuito (il GM la concede), 1→2=15, 2→3=30, 3→4=50, 4→5=80
	upgradeCosts := []int{0, 15, 30, 50, 80}
	currentLv := skill.Level
	if currentLv < 0 {
		currentLv = 0
	}
	var cost int
	if currentLv < len(upgradeCosts) {
		cost = upgradeCosts[currentLv]
	} else {
		cost = 80
	}

	if char.TreePointsAvailable < cost {
		badRequest(c, fmt.Sprintf("punti skill insufficienti (servono %d, hai %d)", cost, char.TreePointsAvailable))
		return
	}

	char.TreePointsAvailable -= cost
	char.CustomSkills[skillIdx].Level = currentLv + 1

	_, err = h.DB.Query(
		"UPDATE $id SET tree_points_available=$pts, custom_skills=$skills",
		map[string]any{
			"id":     char.ID,
			"pts":    char.TreePointsAvailable,
			"skills": char.CustomSkills,
		},
	)
	if err != nil {
		internalError(c, err)
		return
	}

	ok(c, gin.H{
		"skill":                 char.CustomSkills[skillIdx],
		"tree_points_available": char.TreePointsAvailable,
	})
}
