package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"daydream/internal/agents"
	"daydream/internal/db"
	"daydream/internal/llm"
	"daydream/internal/models"
	"daydream/internal/queue"
	"daydream/internal/rag"
	"strings"
)

const maxRetries = 2

// Engine è l'orchestratore del turno di gioco.
type Engine struct {
	db        *db.Client
	llm       llm.Provider
	skills    *SkillRegistry
	retriever *rag.Retriever    // nil se RAG disabilitato
	validator *agents.Validator // nil se non configurato
	compactor *agents.Compactor // nil se non configurato
}

func NewEngine(database *db.Client, provider llm.Provider, skills *SkillRegistry, retriever *rag.Retriever) *Engine {
	return &Engine{db: database, llm: provider, skills: skills, retriever: retriever}
}

// WithAgents configura il Validator e il Compactor Agent (chiamata opzionale dopo NewEngine).
func (e *Engine) WithAgents(v *agents.Validator, c *agents.Compactor) *Engine {
	e.validator = v
	e.compactor = c
	return e
}

// AsTurnProcessor restituisce il Engine come queue.TurnProcessor per iniettarlo nella PlayerQueue.
func (e *Engine) AsTurnProcessor() queue.TurnProcessor {
	return func(ctx context.Context, job queue.TurnJob) {
		e.ProcessTurn(ctx, job)
	}
}

// ProcessTurn è il flusso completo di un turno:
//  1. Carica stato
//  2. Pre-elabora (cooldown, status effects)
//  3. Costruisce prompt
//  4. Chiama LLM con streaming → narrative tokens via SSE
//  5. Parsa risposta JSON
//  6. Applica state_updates + battle_tags
//  7. Post-elabora (level-up, ecc.)
//  8. Salva stato
//  9. Invia evento SSE "done"
func (e *Engine) ProcessTurn(ctx context.Context, job queue.TurnJob) {
	defer close(job.TokenCh)

	// 1. Carica stato
	state, err := e.loadState(job.UserID)
	if err != nil {
		job.TokenCh <- models.SSEEvent{Type: "error", Text: "Errore caricamento stato: " + err.Error()}
		return
	}

	// 2. Pre-elabora
	e.preProcess(state)

	// 3. RAG: recupera contesto rilevante dalla knowledge_base
	var kbEntries []rag.KBEntry
	if e.retriever != nil {
		kbEntries = e.retriever.Retrieve(ctx, job.Input, 4)
	}

	// 3b. Carica world flags rilevanti
	dungeonName := ""
	if state.Session.ActiveDungeon != nil {
		dungeonName = state.Session.ActiveDungeon.Name
	}
	worldFlags := LoadRelevantFlags(e.db, state.Character.ID, state.Session.Location, dungeonName)

	// 4. Costruisce prompt
	msgs := BuildMessages(state, job.Input, e.skills, kbEntries, worldFlags)

	// 4. Chiama LLM con streaming e retry su parse fallito
	var gmResponse *models.GMResponse
	var fullNarrative string

	for attempt := 0; attempt <= maxRetries; attempt++ {
		rawJSON, narrative, streamErr := e.callLLMStreaming(ctx, msgs, job.TokenCh, attempt > 0)
		if streamErr != nil {
			if errors.Is(streamErr, context.Canceled) {
				return
			}
			job.TokenCh <- models.SSEEvent{Type: "error", Text: "Errore LLM: " + streamErr.Error()}
			return
		}

		gmResponse, err = parseGMResponse(rawJSON)
		if err == nil {
			fullNarrative = narrative
			break
		}

		if attempt == maxRetries {
			job.TokenCh <- models.SSEEvent{Type: "error", Text: "Il GM ha risposto con JSON non valido dopo " + fmt.Sprint(maxRetries+1) + " tentativi."}
			return
		}

		// Retry: aggiungo un hint al messaggio
		msgs = append(msgs,
			llmMessage{Role: "assistant", Content: rawJSON},
			llmMessage{Role: "user", Content: "La tua risposta non era JSON valido. Riprova con JSON corretto. " +
				"Inizia con { e finisci con }. Il campo 'narrative' DEVE essere il primo."},
		)
	}

	_ = fullNarrative

	// 5. Applica state_updates
	var stateUpdateEvents []PostEvent
	if gmResponse.StateUpdates != nil {
		stateUpdateEvents = applyStateUpdates(state, gmResponse.StateUpdates, e.skills)
	}
	if gmResponse.ContextMemo != "" {
		state.Session.ContextMemo = gmResponse.ContextMemo
	}
	if len(gmResponse.WorldFlags) > 0 {
		UpsertWorldFlags(e.db, state.Character.ID, gmResponse.WorldFlags)
	}
	if len(gmResponse.CustomSkills) > 0 {
		SaveCustomSkills(e.db, state.Character.ID, gmResponse.CustomSkills)
		// Aggiorna il personaggio in-memory con le nuove skill
		if state.Character.CustomSkills == nil {
			state.Character.CustomSkills = []models.GMCustomSkill{}
		}
		existingIDs := map[string]bool{}
		for _, s := range state.Character.CustomSkills {
			existingIDs[s.ID] = true
		}
		for _, s := range gmResponse.CustomSkills {
			if !existingIDs[s.ID] {
				state.Character.CustomSkills = append(state.Character.CustomSkills, s)
				existingIDs[s.ID] = true
			}
		}
	}

	// 6. Applica battle_tags (snapshot oro prima per il ledger)
	moneyBefore := state.Character.Money
	postEvents := ApplyBattleTags(gmResponse.BattleTags, state, e.skills)

	// Aggiungi eventi da state_updates (incluso reward quest)
	postEvents = append(postEvents, stateUpdateEvents...)

	// 6b. Avanza il clock in-game
	if state.Session.GameTime.Day == 0 && state.Session.GameTime.Hour == 0 {
		// Prima sessione: inizia alle 08:00 del Giorno 1
		state.Session.GameTime = models.GameTime{Day: 1, Hour: 8, Minute: 0}
	}
	elapsed := CalculateTimeElapsed(gmResponse.ActionCategory, gmResponse.BattleTags)
	AdvanceClock(state.Session, elapsed, gmResponse.ActionCategory)

	// 6c. Sleep deprivation
	ApplySleepDeprivation(state.Character, state.Session.HoursAwake)

	// 6d. Quest escalation
	CheckQuestEscalation(state.Session, e.db, state.Character.ID)

	// 6e. Quest balancing: se è stata creata una nuova quest, bilanciala
	if gmResponse.StateUpdates != nil && gmResponse.StateUpdates.Quests != nil &&
		gmResponse.StateUpdates.Quests.Start != nil {
		if len(state.Session.QuestsActive) > 0 {
			lastIdx := len(state.Session.QuestsActive) - 1
			BalanceQuest(&state.Session.QuestsActive[lastIdx], state.Character.Level, state.Session.GameTime)
		}
	}

	// 7. Aggiorna session log
	state.Session.SessionLog = append(state.Session.SessionLog, models.SessionMessage{
		Role:    "player",
		Content: job.Input,
	})
	state.Session.SessionLog = append(state.Session.SessionLog, models.SessionMessage{
		Role:    "gm",
		Content: gmResponse.Narrative,
	})
	if len(state.Session.SessionLog) > 50 {
		state.Session.SessionLog = state.Session.SessionLog[len(state.Session.SessionLog)-50:]
	}

	state.Session.PendingNarrativeEvents = []string{}
	normalizeSession(state.Session)
	state.Session.TurnID++

	// 7b. Tactical tension
	postEvents = append(postEvents, applyTacticalTension(state.Session, postEvents)...)

	// 8. Post-elabora (level-up, loot drop, overdrive, death)
	levelUp := false
	newLevel := 0
	overdrive := false
	playerDied := false
	var lootResult *models.LootResult
	serverUIEvents := []string{}

	for _, ev := range postEvents {
		switch ev.Type {
		case "level_up":
			levelUp = true
			if pl, ok := ev.Payload.(map[string]any); ok {
				if lvl, ok := pl["new_level"].(int); ok {
					newLevel = lvl
				}
			}
			// Direttiva al GM: narra il level-up nel prossimo turno
			state.Session.PendingNarrativeEvents = append(
				state.Session.PendingNarrativeEvents,
				fmt.Sprintf("LEVEL_UP: Il giocatore ha raggiunto il livello %d! Celebra brevemente questo momento con entusiasmo narrativo. HP/MP/STM sono stati ripristinati al massimo. Il giocatore ha %d punti stat da distribuire (PUT /character/stats).",
					newLevel, state.Character.StatPointsAvailable),
			)
		case "loot_drop":
			lootMoneyBefore := state.Character.Money
			loot := GenerateLoot(state.Session.CurrentEnemy, state.Character.Level, state.Character.Stats.LUC, state.Character.Stats.TEC)
			if loot.Gold > 0 {
				state.Character.Money += loot.Gold
				recordGoldDelta(ctx, e.db, state.Character.ID, state.Session.TurnID,
					lootMoneyBefore, state.Character.Money, "loot")
			}
			for _, item := range loot.Items {
				state.Inventory.Bag = append(state.Inventory.Bag, item)
			}
			lootResult = &loot
		case "overdrive":
			overdrive = true
		case "player_dead":
			playerDied = true
			serverUIEvents = append(serverUIEvents, "DEATH")
		case "near_death":
			state.Character.ActionCounters.NearDeathSurvives++
		}
	}

	// Ledger: registra delta oro da battle_tags (include death penalty se presente)
	battleTagSource := "battle_tag"
	if playerDied {
		battleTagSource = "death"
	}
	recordGoldDelta(ctx, e.db, state.Character.ID, state.Session.TurnID,
		moneyBefore, state.Character.Money, battleTagSource)

	// Merge UIEvents del GM con quelli generati server-side
	allUIEvents := append(gmResponse.UIEvents, serverUIEvents...)

	// 9. Salva stato
	if saveErr := e.saveState(state); saveErr != nil {
		job.TokenCh <- models.SSEEvent{Type: "error", Text: "Errore salvataggio: " + saveErr.Error()}
		return
	}

	// 10. Agent post-turno (asincroni, non bloccano la risposta)
	if e.validator != nil {
		e.validator.ValidateAsync(ctx, gmResponse, state)
	}
	if e.compactor != nil {
		e.compactor.CompactIfNeeded(ctx, state.Session.ID, state.Session.ContextMemo)
	}

	// 11. Evento done
	job.TokenCh <- models.SSEEvent{
		Type: "done",
		Payload: models.DonePayload{
			Narrative:  gmResponse.Narrative,
			UIEvents:   allUIEvents,
			Character:  state.Character,
			Inventory:  state.Inventory,
			Session:    state.Session,
			LevelUp:    levelUp,
			NewLevel:   newLevel,
			Loot:       lootResult,
			Overdrive:  overdrive,
			PlayerDied: playerDied,
			GameTime:   state.Session.GameTime,
		},
	}
}

// callLLMStreaming chiama il provider LLM con streaming, emette token narrativi via TokenCh,
// e restituisce il JSON completo accumulato.
// Se isRetry=true non riemette token (l'utente ha già visto la risposta precedente fallita).
func (e *Engine) callLLMStreaming(ctx context.Context, msgs []llmMessage, tokenCh chan<- models.SSEEvent, isRetry bool) (string, string, error) {
	// Converti a llm.Message
	llmMsgs := make([]llm.Message, len(msgs))
	for i, m := range msgs {
		llmMsgs[i] = llm.Message{Role: m.Role, Content: m.Content}
	}

	rawTokens := make(chan string, 256)

	go func() {
		defer close(rawTokens)
		e.llm.Stream(ctx, llmMsgs, llm.CompleteOpts{ //nolint
			JSONMode:  true,
			MaxTokens: 2048,
		}, rawTokens)
	}()

	var fullBuf strings.Builder
	var narrativeBuf strings.Builder
	extractor := &narrativeExtractor{}

	for token := range rawTokens {
		fullBuf.WriteString(token)

		if !isRetry {
			// Estrai token narrativi in real-time
			narrativeChunk := extractor.feed(token)
			if narrativeChunk != "" {
				narrativeBuf.WriteString(narrativeChunk)
				select {
				case tokenCh <- models.SSEEvent{Type: "token", Text: narrativeChunk}:
				case <-ctx.Done():
					return "", "", ctx.Err()
				}
			}
		}
	}

	return fullBuf.String(), narrativeBuf.String(), nil
}

// narrativeExtractor è una FSM che estrae il valore del campo "narrative" da un JSON stream.
type narrativeExtractor struct {
	buf     strings.Builder
	state   int  // 0=searching, 1=in_narrative, 2=done
	escaped bool
}

const narrativeKey = `"narrative":"`

func (ne *narrativeExtractor) feed(token string) string {
	if ne.state == 2 {
		return ""
	}

	var out strings.Builder
	for _, ch := range token {
		ne.buf.WriteRune(ch)

		switch ne.state {
		case 0: // searching for "narrative":"
			if strings.HasSuffix(ne.buf.String(), narrativeKey) {
				ne.state = 1
			}

		case 1: // inside the narrative string value
			if ne.escaped {
				ne.escaped = false
				// Emetti il carattere escaped (es. \n → newline, \" → virgolette)
				switch ch {
				case 'n':
					out.WriteRune('\n')
				case 't':
					out.WriteRune('\t')
				case '"':
					out.WriteRune('"')
				case '\\':
					out.WriteRune('\\')
				default:
					out.WriteRune(ch)
				}
			} else if ch == '\\' {
				ne.escaped = true
			} else if ch == '"' {
				ne.state = 2 // fine del valore narrative
			} else {
				out.WriteRune(ch)
			}
		}
	}

	return out.String()
}

// parseGMResponse tenta di deserializzare il JSON del GM. Pulisce il JSON da
// eventuali markdown code block (```json...```).
func parseGMResponse(raw string) (*models.GMResponse, error) {
	raw = strings.TrimSpace(raw)

	// Rimuovi markdown code block se presente
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) >= 2 {
			// Rimuovi prima e ultima riga (``` e ```)
			lines = lines[1:]
			if lines[len(lines)-1] == "```" {
				lines = lines[:len(lines)-1]
			}
			raw = strings.Join(lines, "\n")
		}
	}

	// Trova l'inizio del JSON
	start := strings.IndexByte(raw, '{')
	end := strings.LastIndexByte(raw, '}')
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("nessun oggetto JSON trovato nella risposta")
	}
	raw = raw[start : end+1]

	var resp models.GMResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("JSON parse: %w", err)
	}
	if resp.Narrative == "" {
		return nil, fmt.Errorf("campo 'narrative' mancante o vuoto")
	}

	return &resp, nil
}

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

// normalizeSession garantisce che tutti i campi slice della sessione siano
// inizializzati come array vuoti (non nil) prima del salvataggio su SurrealDB.
// SurrealDB rifiuta nil/NONE su campi definiti come TYPE array.
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

// applyQuestUpdate gestisce start/complete/fail/progress delle quest.
// Restituisce PostEvent per level_up da exp ricompensa.
func applyQuestUpdate(state *models.FullState, upd *models.QuestsStateUpdate, skills *SkillRegistry) []PostEvent {
	if upd == nil {
		return nil
	}
	sess := state.Session

	var events []PostEvent

	// Start: aggiungi nuova quest
	if upd.Start != nil {
		upd.Start.Status = "active"
		upd.Start.StartedAt = sess.TurnID
		sess.QuestsActive = append(sess.QuestsActive, *upd.Start)
	}

	// Complete / Fail
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

	// Rimuovi da active quelle completate/fallite
	active := sess.QuestsActive[:0]
	for _, q := range sess.QuestsActive {
		if q.Status == "active" {
			active = append(active, q)
		}
	}
	sess.QuestsActive = active

	// Progress
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

// preProcess gestisce le operazioni pre-turno: decremento cooldown, status effects.
func (e *Engine) preProcess(state *models.FullState) {
	// Decrementa cooldown skill di 1 turno
	for k, v := range state.Character.SkillCooldowns {
		if v <= 1 {
			delete(state.Character.SkillCooldowns, k)
		} else {
			state.Character.SkillCooldowns[k] = v - 1
		}
	}

	// Decrementa durata status effects
	active := state.Character.StatusEffects[:0]
	for _, se := range state.Character.StatusEffects {
		se.TurnsRemaining--
		if se.TurnsRemaining > 0 {
			active = append(active, se)
		}
		// TODO Phase 2: applica effetti per-turno (veleno, regen)
	}
	state.Character.StatusEffects = active
}

// loadState carica lo stato completo del personaggio di un utente.
func (e *Engine) loadState(userID string) (*models.FullState, error) {
	// 1. Character
	results, err := e.db.Query(
		"SELECT * FROM character WHERE user_id = $uid",
		map[string]any{"uid": userID},
	)
	if err != nil {
		return nil, fmt.Errorf("load character: %w", err)
	}
	var char models.Character
	if err := results[0].First(&char); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, fmt.Errorf("nessun personaggio trovato")
		}
		return nil, fmt.Errorf("decode character: %w", err)
	}

	// 2. Inventory
	results, err = e.db.Query(
		"SELECT * FROM inventory WHERE character_id = $cid",
		map[string]any{"cid": char.ID},
	)
	if err != nil {
		return nil, fmt.Errorf("load inventory: %w", err)
	}
	var inv models.Inventory
	if err := results[0].First(&inv); err != nil {
		return nil, fmt.Errorf("decode inventory: %w", err)
	}

	// 3. Game session
	results, err = e.db.Query(
		"SELECT * FROM game_session WHERE character_id = $cid",
		map[string]any{"cid": char.ID},
	)
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}
	var sess models.GameSession
	if err := results[0].First(&sess); err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}

	return &models.FullState{
		Character: &char,
		Inventory: &inv,
		Session:   &sess,
	}, nil
}

// saveState persiste lo stato completo su SurrealDB.
func (e *Engine) saveState(state *models.FullState) error {
	if err := e.db.UpdateRecord(state.Character.ID, state.Character); err != nil {
		return fmt.Errorf("save character: %w", err)
	}
	if err := e.db.UpdateRecord(state.Inventory.ID, state.Inventory); err != nil {
		return fmt.Errorf("save inventory: %w", err)
	}
	if err := e.db.UpdateRecord(state.Session.ID, state.Session); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}
