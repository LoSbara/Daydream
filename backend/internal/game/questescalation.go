package game

import (
	"daydream/internal/db"
	"daydream/internal/models"
)

// CheckQuestEscalation controlla tutte le quest attive e avanza
// l'escalation stage se sono state raggiunte le soglie di tempo.
func CheckQuestEscalation(sess *models.GameSession, database db.DBClient, charID string) {
	now := sess.GameTime
	nowMin := now.TotalMinutes()

	for qi := range sess.QuestsActive {
		q := &sess.QuestsActive[qi]
		if q.Status != "active" {
			continue
		}
		if q.DeadlineDay == 0 && q.DeadlineHour == 0 {
			continue
		}

		deadlineMin := q.DeadlineDay*1440 + q.DeadlineHour*60
		if deadlineMin <= 0 {
			continue
		}

		if nowMin >= deadlineMin {
			if q.Status == "active" {
				q.Status = "expired"
				q.CompletedAt = sess.TurnID
				emitQuestFlag(database, charID, q.ID, "expired", "true")
				sess.QuestsCompleted = append(sess.QuestsCompleted, *q)
			}
			continue
		}

		percentConsumed := (nowMin * 100) / deadlineMin

		for _, stage := range q.Escalations {
			if percentConsumed >= stage.TriggerAtPercent && q.EscalationStage < stage.Stage {
				q.EscalationStage = stage.Stage
				if stage.WorldFlagKey != "" {
					emitQuestFlag(database, charID, stage.WorldFlagKey, stage.WorldFlagValue, "world")
				}
			}
		}
	}

	active := sess.QuestsActive[:0]
	for _, q := range sess.QuestsActive {
		if q.Status == "active" {
			active = append(active, q)
		}
	}
	sess.QuestsActive = active
}

// emitQuestFlag upsert un world flag relativo a una quest.
func emitQuestFlag(database db.DBClient, charID, key, value, scope string) {
	if database == nil {
		return
	}
	_, _ = database.Query(
		`UPSERT world_flags SET character_id=$cid, scope=$scope, key=$key, value=$value, updated_at=time::now()
         WHERE character_id=$cid AND scope=$scope AND key=$key`,
		map[string]any{"cid": charID, "scope": scope, "key": key, "value": value},
	)
}
