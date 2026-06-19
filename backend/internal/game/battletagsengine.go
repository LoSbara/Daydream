package game

import (
	"fmt"
	"log/slog"
	"daydream/internal/models"
	"strconv"
	"strings"
)

// PostEvent è un evento post-elaborazione generato dal server (non dal GM).
type PostEvent struct {
	Type    string `json:"type"`    // "level_up" | "player_dead" | "enemy_dead" | "near_death" | "skill_denied" | "loot"
	Payload any    `json:"payload,omitempty"`
}

// ApplyBattleTags processa la lista di battle tags emessa dal GM e modifica
// lo stato in-memory. Restituisce gli eventi post-processing generati.
//
// Tag Phase 1 (risorse e combattimento):
//   PLAYER_HP_±N, PLAYER_MP_±N, PLAYER_STM_±N
//   ENEMY_HP_-N, GOLD_LOSE_N, GOLD_GAIN_N, EXP_GAIN_N
//   PLAYER_DEAD, ENEMY_DEAD, PLAYER_DODGE, PLAYER_CRIT
//
// Tag Phase 2 (skill, status, loot):
//   SKILL_USE_<id>             → usa la skill (verifica costi, applica cooldown)
//   BUFF_<tipo>_<durata>       → es. BUFF_ATK_3 → +ATK per 3 turni
//   DEBUFF_<tipo>_<durata>     → es. DEBUFF_STUN_1 → stun per 1 turno
//   ITEM_USE_<id>              → usa un consumabile dalla borsa
//   LOOT_DROP                  → trigger loot generation server-side (in engine.go)
//   ENEMY_ANALYZE              → analizza il nemico (conta per achievement)
func ApplyBattleTags(tags []string, state *models.FullState, skills *SkillRegistry) []PostEvent {
	var events []PostEvent

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}

		var ev *PostEvent
		var err error

		switch {
		case strings.HasPrefix(tag, "PLAYER_HP_"):
			ev, err = applyResourceDelta(tag, "PLAYER_HP_", &state.Character.Stats.HP)

		case strings.HasPrefix(tag, "PLAYER_MP_"):
			ev, err = applyResourceDelta(tag, "PLAYER_MP_", &state.Character.Stats.MP)

		case strings.HasPrefix(tag, "PLAYER_STM_"):
			ev, err = applyResourceDelta(tag, "PLAYER_STM_", &state.Character.Stats.STM)

		case strings.HasPrefix(tag, "ENEMY_HP_"):
			if state.Session.CurrentEnemy != nil {
				delta, parseErr := parseDelta(strings.TrimPrefix(tag, "ENEMY_HP_"))
				if parseErr != nil {
					break
				}
				enemy := state.Session.CurrentEnemy
				prevHP := enemy.HP
				enemy.HP = Clamp(enemy.HP+delta, 0, enemy.MaxHP)
				if enemy.HP <= 0 {
					enemy.HP = 0
				}
				// Boss phase transition: solo su danni (delta < 0) a nemici tier "boss"
				if delta < 0 && enemy.Tier == "boss" && enemy.MaxHP > 0 {
					hpPct := enemy.HP * 100 / enemy.MaxHP
					prevPct := prevHP * 100 / enemy.MaxHP
					if enemy.CurrentPhase < 2 && hpPct <= 50 && prevPct > 50 {
						enemy.CurrentPhase = 2
						events = append(events, PostEvent{
							Type:    "boss_phase_change",
							Payload: map[string]any{"phase": 2, "enemy": enemy.Name},
						})
						state.Session.PendingNarrativeEvents = append(
							state.Session.PendingNarrativeEvents,
							fmt.Sprintf("[BOSS_PHASE_2] %s entra in Fase 2 — comportamento cambiato, nuovi attacchi attivi", enemy.Name),
						)
					}
					if enemy.CurrentPhase < 3 && hpPct <= 25 && prevPct > 25 {
						enemy.CurrentPhase = 3
						events = append(events, PostEvent{
							Type:    "boss_phase_change",
							Payload: map[string]any{"phase": 3, "enemy": enemy.Name},
						})
						state.Session.PendingNarrativeEvents = append(
							state.Session.PendingNarrativeEvents,
							fmt.Sprintf("[BOSS_PHASE_3] %s entra in Fase 3 — modalità disperata, massima potenza", enemy.Name),
						)
					}
				}
			}

		case strings.HasPrefix(tag, "GOLD_LOSE_"):
			amount, parseErr := strconv.Atoi(strings.TrimPrefix(tag, "GOLD_LOSE_"))
			if parseErr != nil {
				break
			}
			// Layer 2 — cooldown anti-duplicazione: blocca se siamo ancora nel periodo di protezione
			if state.Session.TurnID < state.Session.GoldLoseCooldownUntil {
				slog.Warn("GOLD_LOSE bloccato dal cooldown anti-duplicazione",
					"tag", tag,
					"turn", state.Session.TurnID,
					"cooldown_until", state.Session.GoldLoseCooldownUntil)
				events = append(events, PostEvent{Type: "gold_blocked", Payload: amount})
				break
			}
			state.Character.Money = Clamp(state.Character.Money-amount, 0, 99999999)
			// Layer 2: imposta cooldown per i prossimi 2 turni
			state.Session.GoldLoseCooldownUntil = state.Session.TurnID + 2
			// Layer 1: aggiorna la nota di transazione visibile al GM nel prossimo turno
			state.Session.LastGoldTransaction = fmt.Sprintf(
				"ACQUISTO turno %d: -%d oro → saldo %d [TRANSAZIONE CHIUSA — non emettere altri GOLD_LOSE per questo acquisto]",
				state.Session.TurnID, amount, state.Character.Money)

		case strings.HasPrefix(tag, "GOLD_GAIN_"):
			amount, parseErr := strconv.Atoi(strings.TrimPrefix(tag, "GOLD_GAIN_"))
			if parseErr != nil {
				break
			}
			state.Character.Money += amount
			if state.Character.Money > state.Character.ActionCounters.MaxMoney {
				state.Character.ActionCounters.MaxMoney = state.Character.Money
			}
			state.Session.LastGoldTransaction = fmt.Sprintf(
				"GUADAGNO turno %d: +%d oro → saldo %d",
				state.Session.TurnID, amount, state.Character.Money)

		case strings.HasPrefix(tag, "EXP_GAIN_"):
			amount, parseErr := strconv.Atoi(strings.TrimPrefix(tag, "EXP_GAIN_"))
			if parseErr != nil {
				break
			}
			levelUpEv := applyExp(state.Character, amount)
			if levelUpEv != nil {
				events = append(events, *levelUpEv)
			}

		case tag == "PLAYER_DEAD":
			ev = applyPlayerDead(state)

		case tag == "ENEMY_DEAD":
			ev = applyEnemyDead(state)

		case tag == "PLAYER_DODGE":
			state.Character.ActionCounters.Dodges++

		case tag == "PLAYER_CRIT":
			state.Character.ActionCounters.Criticals++

		case tag == "ENEMY_ANALYZE":
			if state.Session.CurrentEnemy != nil {
				state.Character.ActionCounters.EnemiesAnalyzed++
			}

		// Phase 2: SKILL_USE_<skill_id>
		case strings.HasPrefix(tag, "SKILL_USE_"):
			skillID := strings.TrimPrefix(tag, "SKILL_USE_")
			skillEv := applySkillUse(skillID, state, skills)
			if skillEv != nil {
				events = append(events, *skillEv)
			}

		// Phase 2: BUFF_<tipo>_<durata>
		case strings.HasPrefix(tag, "BUFF_"):
			applyStatusTag(tag, "buff", state.Character)

		// Phase 2: DEBUFF_<tipo>_<durata>
		case strings.HasPrefix(tag, "DEBUFF_"):
			applyStatusTag(tag, "debuff", state.Character)

		// Phase 2: ITEM_USE_<id>
		case strings.HasPrefix(tag, "ITEM_USE_"):
			itemID := strings.TrimPrefix(tag, "ITEM_USE_")
			applyItemUse(itemID, state)

		// Phase 2: LOOT_DROP — segnala al caller di generare loot
		case tag == "LOOT_DROP":
			events = append(events, PostEvent{Type: "loot_drop"})

		// Navigazione dungeon — emesso dal GM quando il giocatore si sposta
		case strings.HasPrefix(tag, "DUNGEON_MOVE_"):
			direction := strings.TrimPrefix(tag, "DUNGEON_MOVE_")
			ev = applyDungeonMove(direction, state)
		}

		if err != nil {
			_ = fmt.Errorf("battle tag %q: %w", tag, err)
		}
		if ev != nil {
			events = append(events, *ev)
		}
	}

	// Near death: HP ≤ 20% del max
	if state.Character.Stats.HP.Current > 0 &&
		state.Character.Stats.HP.Current <= state.Character.Stats.HP.Max/5 {
		events = append(events, PostEvent{Type: "near_death"})
	}

	return events
}

// applySkillUse verifica costi/cooldown e applica la skill.
// Restituisce un PostEvent "skill_denied" se la skill non può essere usata.
func applySkillUse(skillID string, state *models.FullState, skills *SkillRegistry) *PostEvent {
	if skills == nil {
		return nil
	}
	skill := skills.Get(skillID)
	if skill == nil {
		return &PostEvent{Type: "skill_denied", Payload: map[string]any{
			"skill_id": skillID, "reason": "skill non trovata",
		}}
	}

	char := state.Character

	// Controlla classe
	if skill.Job != char.Job {
		return &PostEvent{Type: "skill_denied", Payload: map[string]any{
			"skill_id": skillID, "reason": "classe sbagliata",
		}}
	}

	// Controlla sblocco
	if !skills.IsUnlocked(skill, char) {
		return &PostEvent{Type: "skill_denied", Payload: map[string]any{
			"skill_id": skillID, "reason": "skill non ancora sbloccata",
		}}
	}

	// Controlla cooldown
	if remaining, ok := char.SkillCooldowns[skillID]; ok && remaining > 0 {
		return &PostEvent{Type: "skill_denied", Payload: map[string]any{
			"skill_id": skillID, "reason": fmt.Sprintf("in cooldown (%d turni)", remaining),
		}}
	}

	// Controlla risorse
	if char.Stats.MP.Current < skill.MPCost {
		return &PostEvent{Type: "skill_denied", Payload: map[string]any{
			"skill_id": skillID, "reason": "MP insufficienti",
		}}
	}
	if char.Stats.STM.Current < skill.STMCost {
		return &PostEvent{Type: "skill_denied", Payload: map[string]any{
			"skill_id": skillID, "reason": "Stamina insufficiente",
		}}
	}

	// Applica costi
	char.Stats.MP.Current -= skill.MPCost
	char.Stats.STM.Current -= skill.STMCost

	// Imposta cooldown
	if skill.CooldownTurns > 0 {
		char.SkillCooldowns[skillID] = skill.CooldownTurns
	}

	// Traccia per achievement (max skill usate in combattimento non è direttamente qui)
	return nil // successo, nessun PostEvent speciale
}

// applyStatusTag analizza "BUFF_tipo_durata" o "DEBUFF_tipo_durata" e aggiunge lo StatusEffect.
// Formato: BUFF_ATK_3, DEBUFF_STUN_1, BUFF_SHIELD_1, ecc.
func applyStatusTag(tag, seType string, char *models.Character) {
	prefix := "BUFF_"
	if seType == "debuff" {
		prefix = "DEBUFF_"
	}
	rest := strings.TrimPrefix(tag, prefix)

	// rest = "ATK_3" o "STUN_1"
	lastUnderscore := strings.LastIndex(rest, "_")
	if lastUnderscore < 0 {
		return
	}
	typeName := rest[:lastUnderscore]
	durationStr := rest[lastUnderscore+1:]
	duration, err := strconv.Atoi(durationStr)
	if err != nil || duration <= 0 {
		return
	}

	// Rimuovi status dello stesso tipo se già presente
	existing := char.StatusEffects[:0]
	for _, se := range char.StatusEffects {
		if se.ID != tag {
			existing = append(existing, se)
		}
	}
	char.StatusEffects = existing

	icon, color := statusEffectStyle(strings.ToUpper(typeName), seType)

	char.StatusEffects = append(char.StatusEffects, models.StatusEffect{
		ID:             tag,
		Name:           typeName,
		Icon:           icon,
		Type:           seType,
		TurnsRemaining: duration,
		Value:          duration,
		Color:          color,
	})
}

// applyItemUse rimuove un consumabile dalla borsa e applica il suo effetto.
func applyItemUse(itemID string, state *models.FullState) {
	bag := state.Inventory.Bag
	for i, item := range bag {
		if item.ID == itemID && item.Type == "consumable" {
			// Applica effetto base (HP_POTION → +30 HP)
			// TODO Phase 3: effetti più complessi
			if strings.Contains(strings.ToLower(item.Name), "pozi") ||
				strings.Contains(strings.ToLower(item.Name), "cura") {
				state.Character.Stats.HP.Current = Clamp(
					state.Character.Stats.HP.Current+30,
					0, state.Character.Stats.HP.Max,
				)
			}
			// Rimuovi dalla borsa (o decrementa quantità)
			if item.Quantity > 1 {
				bag[i].Quantity--
			} else {
				bag = append(bag[:i], bag[i+1:]...)
			}
			state.Inventory.Bag = bag
			return
		}
	}
}

// applyTacticalTension gestisce l'incremento della tension e il trigger overdrive.
// Va chiamata dopo ApplyBattleTags così che enemy_dead/player_dead abbiano già azzerato la tension.
func applyTacticalTension(sess *models.GameSession, events []PostEvent) []PostEvent {
	// Se il combattimento è già finito (enemy_dead/player_dead già eseguiti) non incrementare.
	for _, ev := range events {
		if ev.Type == "enemy_dead" || ev.Type == "player_dead" {
			return nil
		}
	}

	if !sess.CombatActive {
		// Fuori combattimento: decrementa lentamente verso zero
		if sess.TacticalTension > 0 {
			sess.TacticalTension = Clamp(sess.TacticalTension-5, 0, 100)
		}
		return nil
	}

	wasBelow80 := sess.TacticalTension < 80
	sess.TacticalTension = Clamp(sess.TacticalTension+10, 0, 100)

	// Trigger overdrive quando si attraversa la soglia 80
	if wasBelow80 && sess.TacticalTension >= 80 {
		return []PostEvent{{Type: "overdrive"}}
	}
	return nil
}

// ---- helper funcs (anche usate da engine.go) ----

func applyResourceDelta(tag, prefix string, res *models.Resource) (*PostEvent, error) {
	delta, err := parseDelta(strings.TrimPrefix(tag, prefix))
	if err != nil {
		return nil, err
	}
	res.Current = Clamp(res.Current+delta, 0, res.Max)
	return nil, nil
}

func parseDelta(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("delta vuoto")
	}
	return strconv.Atoi(s)
}

func applyExp(char *models.Character, amount int) *PostEvent {
	char.Experience += amount
	if char.Experience < char.ExperienceToNext {
		return nil
	}

	char.Experience -= char.ExperienceToNext
	char.Level++
	char.ExperienceToNext = ExperienceToNextLevel(char.Level)
	char.StatPointsAvailable += 5
	char.TreePointsAvailable += 3
	char.SkillSlots++

	// Ripristino completo HP/MP/STM al level-up (convenzione JRPG standard)
	char.Stats.HP.Current = char.Stats.HP.Max
	char.Stats.MP.Current = char.Stats.MP.Max
	char.Stats.STM.Current = char.Stats.STM.Max

	return &PostEvent{
		Type:    "level_up",
		Payload: map[string]any{"new_level": char.Level},
	}
}

func applyPlayerDead(state *models.FullState) *PostEvent {
	state.Character.Stats.HP.Current = state.Character.Stats.HP.Max / 2
	state.Character.Stats.MP.Current = state.Character.Stats.MP.Max / 2
	state.Character.Stats.STM.Current = state.Character.Stats.STM.Max / 2

	penalty := state.Character.Money / 10
	state.Character.Money = Clamp(state.Character.Money-penalty, 0, 99999999)

	state.Session.CombatActive = false
	state.Session.CurrentEnemy = nil
	state.Session.TacticalTension = 0
	state.Session.GameState = models.StateWorldNavigation
	state.Session.Location = "Nexus"
	state.Session.SubLocation = ""
	state.Session.ZoneType = "safe_zone"

	return &PostEvent{
		Type:    "player_dead",
		Payload: map[string]any{"gold_lost": penalty},
	}
}

func applyDungeonMove(direction string, state *models.FullState) *PostEvent {
	dungeon := state.Session.ActiveDungeon
	if dungeon == nil {
		return nil
	}
	if state.Session.CombatActive {
		return &PostEvent{Type: "dungeon_move_denied", Payload: map[string]any{
			"reason": "impossibile muoversi durante un combattimento",
		}}
	}

	currRoom, ok := dungeon.Rooms[dungeon.CurrentRoom]
	if !ok {
		return nil
	}

	nextID, ok := currRoom.Exits[direction]
	if !ok {
		return &PostEvent{Type: "dungeon_move_denied", Payload: map[string]any{
			"reason": fmt.Sprintf("nessuna uscita verso %q dalla stanza corrente", direction),
		}}
	}

	nextRoom := dungeon.Rooms[nextID]
	nextRoom.Visited = true
	dungeon.Rooms[nextID] = nextRoom
	dungeon.CurrentRoom = nextID
	state.Session.SubLocation = nextRoom.Name

	if nextRoom.HasEnemy && !nextRoom.Cleared {
		state.Session.CombatActive = true
		state.Session.GameState = models.StateDungeonCombat
	} else {
		state.Session.GameState = models.StateDungeonExplore
	}

	return &PostEvent{
		Type:    "dungeon_move",
		Payload: map[string]any{"room_id": nextRoom.ID, "room_name": nextRoom.Name},
	}
}

func applyEnemyDead(state *models.FullState) *PostEvent {
	enemyName := ""
	if state.Session.CurrentEnemy != nil {
		enemyName = state.Session.CurrentEnemy.Name
		state.Character.ActionCounters.EnemiesDefeated++
		if state.Session.CurrentEnemy.Tier == "elite" || state.Session.CurrentEnemy.Tier == "boss" {
			state.Character.ActionCounters.EliteKills++
		}
	}

	state.Session.CombatActive = false
	state.Session.CurrentEnemy = nil
	state.Session.TacticalTension = 0

	if state.Session.GameState == models.StateCombat {
		state.Session.GameState = models.StateWorldNavigation
	} else if state.Session.GameState == models.StateDungeonCombat {
		state.Session.GameState = models.StateDungeonExplore
		// Marca la stanza corrente come liberata
		if state.Session.ActiveDungeon != nil {
			curID := state.Session.ActiveDungeon.CurrentRoom
			if room, ok := state.Session.ActiveDungeon.Rooms[curID]; ok {
				room.Cleared = true
				state.Session.ActiveDungeon.Rooms[curID] = room
			}
		}
	}

	return &PostEvent{
		Type:    "enemy_dead",
		Payload: map[string]any{"enemy": enemyName},
	}
}

// statusEffectStyle ritorna icona e colore per uno status effect in base al tipo.
func statusEffectStyle(name, seType string) (icon, color string) {
	switch name {
	case "POISON":
		return "☠", "#7ec850"
	case "BLEED":
		return "🩸", "#cc3333"
	case "BURN":
		return "🔥", "#e07020"
	case "STUN":
		return "⚡", "#e0c030"
	case "SLOW":
		return "🐌", "#8888cc"
	case "BLIND":
		return "👁", "#666688"
	case "REGEN", "REGEN_HP":
		return "💚", "#40cc70"
	case "REGEN_MP":
		return "💙", "#4080cc"
	case "REGEN_STM":
		return "🟡", "#ccaa20"
	case "ATK":
		return "⚔", "#e04040"
	case "DEF":
		return "🛡", "#6090cc"
	case "SHIELD":
		return "🔵", "#4060bb"
	case "HASTE":
		return "💨", "#80e0e0"
	default:
		if seType == "debuff" {
			return "💀", "#ff6b6b"
		}
		return "✨", "#90ee90"
	}
}
