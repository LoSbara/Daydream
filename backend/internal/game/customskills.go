package game

import (
	"fmt"
	"log/slog"
	"strings"

	"daydream/internal/db"
	"daydream/internal/models"
)

// SaveCustomSkills aggiunge nuove custom skill al personaggio (no duplicati per ID).
func SaveCustomSkills(database *db.Client, charID string, newSkills []models.GMCustomSkill) {
	// Prima carica le skill esistenti
	qr, err := database.QueryOne(
		"SELECT custom_skills FROM character WHERE id=$id",
		map[string]any{"id": charID},
	)
	if err != nil {
		slog.Error("custom skills load failed", "err", err)
		return
	}

	type partial struct {
		CustomSkills []models.GMCustomSkill `json:"custom_skills"`
	}
	var chars []partial
	if err := qr.All(&chars); err != nil || len(chars) == 0 {
		return
	}

	existing := chars[0].CustomSkills
	existingIDs := map[string]bool{}
	for _, s := range existing {
		existingIDs[s.ID] = true
	}

	// Aggiunge solo le skill nuove
	merged := existing
	for _, s := range newSkills {
		if !existingIDs[s.ID] {
			merged = append(merged, s)
			existingIDs[s.ID] = true
			slog.Info("custom skill granted", "char", charID, "skill", s.Name)
		}
	}

	_, err = database.Query(
		"UPDATE $id SET custom_skills=$skills",
		map[string]any{"id": charID, "skills": merged},
	)
	if err != nil {
		slog.Error("custom skills save failed", "err", err)
	}
}

// FormatCustomSkillsForPrompt formatta le custom skill per il GM.
func FormatCustomSkillsForPrompt(skills []models.GMCustomSkill) string {
	if len(skills) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## ABILITÀ UNICHE (concesse narrativamente)\n")
	for _, s := range skills {
		lv := s.Level
		if lv <= 0 {
			lv = 1
		}
		maxLv := s.MaxLevel
		if maxLv == 0 {
			maxLv = 5
		}
		if s.Type == "active" {
			cost := ""
			if s.MPCost > 0 {
				cost += fmt.Sprintf("MP:%d ", s.MPCost)
			}
			if s.STMCost > 0 {
				cost += fmt.Sprintf("STM:%d ", s.STMCost)
			}
			if s.Cooldown > 0 {
				cost += fmt.Sprintf("CD:%dt", s.Cooldown)
			}
			sb.WriteString(fmt.Sprintf("- **%s (Lv%d/%d)** [Attivo | %s] *(Origine: %s)*: %s\n",
				s.Name, lv, maxLv, strings.TrimSpace(cost), s.Origin, s.EffectDesc))
		} else {
			sb.WriteString(fmt.Sprintf("- **%s (Lv%d/%d)** [Passivo] *(Origine: %s)*: %s\n",
				s.Name, lv, maxLv, s.Origin, s.EffectDesc))
		}
	}
	return sb.String()
}
