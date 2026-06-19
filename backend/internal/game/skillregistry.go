package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"daydream/internal/models"
)

// SkillRegistry carica il catalogo skill da disco e lo tiene in memoria.
// È read-only e thread-safe dopo Init().
type SkillRegistry struct {
	skills   map[string]*models.Skill // keyed by skill.ID
	byJob    map[string][]*models.Skill
}

// NewSkillRegistry carica skills.json e costruisce gli indici in memoria.
func NewSkillRegistry() (*SkillRegistry, error) {
	path := skillsJSONPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skills.json: %w", err)
	}

	var list []models.Skill
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parse skills.json: %w", err)
	}

	reg := &SkillRegistry{
		skills: make(map[string]*models.Skill, len(list)),
		byJob:  make(map[string][]*models.Skill),
	}
	for i := range list {
		s := &list[i]
		reg.skills[s.ID] = s
		reg.byJob[s.Job] = append(reg.byJob[s.Job], s)
	}
	return reg, nil
}

// Get restituisce la skill con l'ID dato, o nil se non esiste.
func (r *SkillRegistry) Get(id string) *models.Skill {
	return r.skills[id]
}

// ForJob restituisce tutte le skill disponibili per una classe.
func (r *SkillRegistry) ForJob(job string) []*models.Skill {
	return r.byJob[job]
}

// All restituisce tutte le skill del catalogo.
func (r *SkillRegistry) All() []*models.Skill {
	out := make([]*models.Skill, 0, len(r.skills))
	for _, s := range r.skills {
		out = append(out, s)
	}
	return out
}

// IsUnlocked controlla se il personaggio ha sbloccato una skill data
// (verifica le unlock_condition sui suoi action_counters).
func (r *SkillRegistry) IsUnlocked(skill *models.Skill, char *models.Character) bool {
	if skill.UnlockCondition == nil {
		return true // skill base, sempre disponibile
	}
	cond := skill.UnlockCondition
	ac := char.ActionCounters

	if v, ok := cond["enemies_defeated"]; ok {
		req := toInt(v)
		if ac.EnemiesDefeated < req {
			return false
		}
	}
	if v, ok := cond["dodges"]; ok {
		if ac.Dodges < toInt(v) {
			return false
		}
	}
	if v, ok := cond["near_death_survives"]; ok {
		if ac.NearDeathSurvives < toInt(v) {
			return false
		}
	}
	if v, ok := cond["enemies_analyzed"]; ok {
		if ac.EnemiesAnalyzed < toInt(v) {
			return false
		}
	}
	if v, ok := cond["criticals"]; ok {
		if ac.Criticals < toInt(v) {
			return false
		}
	}
	return true
}

func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

func skillsJSONPath() string {
	if env := os.Getenv("SKILLS_JSON_PATH"); env != "" {
		return env
	}
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "configs", "seeds", "skills.json")
}
