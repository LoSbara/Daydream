package game

import "daydream/internal/models"

// applyStateUpdates applica gli aggiornamenti di stato proposti dal GM allo stato in-memory.
func applyStateUpdates(state *models.FullState, upd *models.StateUpdate, skills *SkillRegistry) []PostEvent {
	if upd == nil {
		return nil
	}
	var events []PostEvent

	if upd.Player != nil {
		p := upd.Player
		if p.Stats != nil {
			applyStatsUpdate(state, p.Stats)
		}
		if p.StatusEffects != nil {
			state.Character.StatusEffects = p.StatusEffects
		}
		if p.Reputation != nil {
			state.Character.Reputation = *p.Reputation
		}
	}

	if upd.GameState != nil {
		ApplyGameState(state.Session, upd.GameState)
	}

	if upd.Quests != nil {
		evts := applyQuestUpdate(state, upd.Quests, skills)
		events = append(events, evts...)
	}

	return events
}

// applyStatsUpdate applica patch parziali alle statistiche del personaggio.
func applyStatsUpdate(state *models.FullState, s *models.StatsUpdate) {
	if s.HP != nil {
		if s.HP.Current != nil {
			state.Character.Stats.HP.Current = Clamp(*s.HP.Current, 0, state.Character.Stats.HP.Max)
		}
		if s.HP.Max != nil {
			state.Character.Stats.HP.Max = *s.HP.Max
		}
	}
	if s.MP != nil {
		if s.MP.Current != nil {
			state.Character.Stats.MP.Current = Clamp(*s.MP.Current, 0, state.Character.Stats.MP.Max)
		}
		if s.MP.Max != nil {
			state.Character.Stats.MP.Max = *s.MP.Max
		}
	}
	if s.STM != nil {
		if s.STM.Current != nil {
			state.Character.Stats.STM.Current = Clamp(*s.STM.Current, 0, state.Character.Stats.STM.Max)
		}
		if s.STM.Max != nil {
			state.Character.Stats.STM.Max = *s.STM.Max
		}
	}
}

// applyQuestUpdate gestisce start/complete/fail/progress delle quest.
// Restituisce PostEvent per level_up da exp ricompensa.
func applyQuestUpdate(state *models.FullState, upd *models.QuestsStateUpdate, skills *SkillRegistry) []PostEvent {
	if upd == nil {
		return nil
	}
	sess := state.Session

	var events []PostEvent

	if upd.Start != nil {
		upd.Start.Status = "active"
		upd.Start.StartedAt = sess.TurnID
		sess.QuestsActive = append(sess.QuestsActive, *upd.Start)
	}

	for i := range sess.QuestsActive {
		q := &sess.QuestsActive[i]
		if q.Status != "active" {
			continue
		}

		if upd.Complete != "" && q.ID == upd.Complete {
			q.Status = "completed"
			q.CompletedAt = sess.TurnID
			evts := applyQuestRewards(state, q, skills)
			events = append(events, evts...)
			sess.QuestsCompleted = append(sess.QuestsCompleted, *q)
		}

		if upd.Fail != "" && q.ID == upd.Fail {
			q.Status = "failed"
			q.CompletedAt = sess.TurnID
			sess.QuestsCompleted = append(sess.QuestsCompleted, *q)
		}
	}

	active := sess.QuestsActive[:0]
	for _, q := range sess.QuestsActive {
		if q.Status == "active" {
			active = append(active, q)
		}
	}
	sess.QuestsActive = active

	for _, prog := range upd.Progress {
		for i := range sess.QuestsActive {
			if sess.QuestsActive[i].ID != prog.QuestID {
				continue
			}
			q := &sess.QuestsActive[i]
			if prog.ObjIndex >= 0 && prog.ObjIndex < len(q.Objectives) {
				obj := &q.Objectives[prog.ObjIndex]
				obj.Current = Clamp(obj.Current+prog.Delta, 0, obj.Required)
				obj.Done = obj.Current >= obj.Required
			}
		}
	}

	return events
}

// applyQuestRewards applica gold, exp e item della quest al personaggio.
func applyQuestRewards(state *models.FullState, q *models.Quest, skills *SkillRegistry) []PostEvent {
	_ = skills
	var events []PostEvent

	if q.Rewards.Gold > 0 {
		state.Character.Money += q.Rewards.Gold
	}

	if q.Rewards.Exp > 0 {
		ev := applyExp(state.Character, q.Rewards.Exp)
		if ev != nil {
			events = append(events, *ev)
		}
	}

	for _, item := range q.Rewards.Items {
		state.Inventory.Bag = append(state.Inventory.Bag, item)
	}

	return events
}

// normalizeSession garantisce che tutti i campi slice della sessione siano
// inizializzati come array vuoti (non nil) prima del salvataggio su SurrealDB.
func normalizeSession(sess *models.GameSession) {
	if sess.SkillLoadout == nil {
		sess.SkillLoadout = []string{}
	}
	if sess.SessionLog == nil {
		sess.SessionLog = []models.SessionMessage{}
	}
	if sess.QuestsActive == nil {
		sess.QuestsActive = []models.Quest{}
	}
	if sess.QuestsCompleted == nil {
		sess.QuestsCompleted = []models.Quest{}
	}
}
