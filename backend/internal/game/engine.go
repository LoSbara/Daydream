package game

import (
	"context"
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
	db           db.DBClient
	llm          llm.Provider
	skills       *SkillRegistry
	retriever    *rag.Retriever         // nil se RAG disabilitato
	validator    *agents.Validator      // nil se non configurato
	compactor    *agents.Compactor      // nil se non configurato
	contentGen   *agents.ContentGenerator // nil se RAG disabilitato
}

func NewEngine(database db.DBClient, provider llm.Provider, skills *SkillRegistry, retriever *rag.Retriever) *Engine {
	return &Engine{db: database, llm: provider, skills: skills, retriever: retriever}
}

// WithAgents configura il Validator e il Compactor Agent (chiamata opzionale dopo NewEngine).
func (e *Engine) WithAgents(v *agents.Validator, c *agents.Compactor) *Engine {
	e.validator = v
	e.compactor = c
	return e
}

// WithContentGenerator configura il Content Generator Agent (richiede RAG abilitato).
func (e *Engine) WithContentGenerator(cg *agents.ContentGenerator) *Engine {
	e.contentGen = cg
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

	// 2. Pre-elabora (cooldown, status effect tick)
	preEvents := e.preProcess(state)

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
		memo := gmResponse.ContextMemo
		if len(memo) > 8000 {
			memo = memo[:8000]
		}
		state.Session.ContextMemo = memo
	}
	if len(gmResponse.WorldFlags) > 0 {
		UpsertWorldFlags(e.db, state.Character.ID, gmResponse.WorldFlags)
	}
	if len(gmResponse.CustomSkills) > 0 {
		// Dedup: elimina duplicati all'interno della stessa risposta GM prima di salvare
		seen := map[string]bool{}
		unique := gmResponse.CustomSkills[:0]
		for _, s := range gmResponse.CustomSkills {
			if s.ID != "" && !seen[s.ID] {
				seen[s.ID] = true
				unique = append(unique, s)
			}
		}
		gmResponse.CustomSkills = unique

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

	// Aggiungi eventi da pre-process (morte da veleno/emorragia) e state_updates
	postEvents = append(postEvents, preEvents...)
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
		case "gold_blocked":
			serverUIEvents = append(serverUIEvents, "GOLD_LOSE_BLOCKED")
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
	if e.contentGen != nil && len(gmResponse.ContentGen) > 0 {
		e.contentGen.GenerateAsync(ctx, gmResponse.ContentGen, state.Character.ID)
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

// preProcess gestisce le operazioni pre-turno: decremento cooldown e tick degli status effect.
// Ritorna PostEvent per morti da veleno/emorragia.
func (e *Engine) preProcess(state *models.FullState) []PostEvent {
	// Decrementa cooldown skill di 1 turno
	for k, v := range state.Character.SkillCooldowns {
		if v <= 1 {
			delete(state.Character.SkillCooldowns, k)
		} else {
			state.Character.SkillCooldowns[k] = v - 1
		}
	}

	// Tick status effect: applica l'effetto, poi decrementa la durata
	active := state.Character.StatusEffects[:0]
	for _, se := range state.Character.StatusEffects {
		tickStatusEffect(&state.Character.Stats, se)
		se.TurnsRemaining--
		if se.TurnsRemaining > 0 {
			active = append(active, se)
		}
	}
	state.Character.StatusEffects = active

	// Morte da status effect (veleno, emorragia, bruciatura)
	if state.Character.Stats.HP.Current <= 0 {
		state.Character.Stats.HP.Current = 0
		state.Session.PendingNarrativeEvents = append(
			state.Session.PendingNarrativeEvents,
			"MORTE_STATUS_EFFECT: Il personaggio ha raggiunto 0 HP a causa di un effetto di stato (veleno/emorragia/bruciatura). Narra la morte in modo drammatico. Ignora l'azione che il giocatore stava tentando di fare.",
		)
		return []PostEvent{{Type: "player_dead"}}
	}
	return nil
}

// tickStatusEffect applica l'effetto per-turno di uno status effect alle statistiche del personaggio.
// Gli effetti scalano su maxHP/maxMP/maxSTM per rimanere bilanciati dall'Lv 1 al 100.
// Effetti narrativi puri (ATK, DEF, SHIELD, STUN, SLOW, BLIND) non hanno tick server-side.
func tickStatusEffect(stats *models.Stats, se models.StatusEffect) {
	switch strings.ToUpper(se.Name) {
	case "POISON":
		dmg := max(1, stats.HP.Max*5/100)
		stats.HP.Current = Clamp(stats.HP.Current-dmg, 0, stats.HP.Max)
	case "BLEED":
		dmg := max(1, stats.HP.Max*8/100)
		stats.HP.Current = Clamp(stats.HP.Current-dmg, 0, stats.HP.Max)
	case "BURN":
		dmg := max(1, stats.HP.Max*6/100)
		stats.HP.Current = Clamp(stats.HP.Current-dmg, 0, stats.HP.Max)
		mpDrain := max(1, stats.MP.Max*4/100)
		stats.MP.Current = Clamp(stats.MP.Current-mpDrain, 0, stats.MP.Max)
	case "REGEN", "REGEN_HP":
		heal := max(1, stats.HP.Max*8/100)
		stats.HP.Current = Clamp(stats.HP.Current+heal, 0, stats.HP.Max)
	case "REGEN_MP":
		regen := max(1, stats.MP.Max*10/100)
		stats.MP.Current = Clamp(stats.MP.Current+regen, 0, stats.MP.Max)
	case "REGEN_STM":
		regen := max(1, stats.STM.Max*10/100)
		stats.STM.Current = Clamp(stats.STM.Current+regen, 0, stats.STM.Max)
	}
}

// loadState carica lo stato completo del personaggio di un utente.
func (e *Engine) loadState(userID string) (*models.FullState, error) {
	// 1. Character
	charQR, err := e.db.QueryOne(
		"SELECT * FROM character WHERE user_id = $uid",
		map[string]any{"uid": userID},
	)
	if err != nil {
		return nil, fmt.Errorf("load character: %w", err)
	}
	var char models.Character
	if err := charQR.First(&char); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, fmt.Errorf("nessun personaggio trovato")
		}
		return nil, fmt.Errorf("decode character: %w", err)
	}

	// 2. Inventory
	invQR, err := e.db.QueryOne(
		"SELECT * FROM inventory WHERE character_id = $cid",
		map[string]any{"cid": char.ID},
	)
	if err != nil {
		return nil, fmt.Errorf("load inventory: %w", err)
	}
	var inv models.Inventory
	if err := invQR.First(&inv); err != nil {
		return nil, fmt.Errorf("decode inventory: %w", err)
	}

	// 3. Game session
	sessQR, err := e.db.QueryOne(
		"SELECT * FROM game_session WHERE character_id = $cid",
		map[string]any{"cid": char.ID},
	)
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}
	var sess models.GameSession
	if err := sessQR.First(&sess); err != nil {
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
