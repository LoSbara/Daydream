package game

import (
	"encoding/json"
	"fmt"
	"daydream/internal/models"
	"daydream/internal/rag"
	"strings"
)

const maxSessionLogTurns = 10

// BuildMessages costruisce la slice di messaggi per l'LLM.
// Architettura a 3 livelli per massimizzare la cache hit:
//   Livello 1 (system): regole statiche → cache 100%
//   Livello 2 (user):   scheda personaggio, loadout, zona → cache 70-90%
//   Livello 3 (user):   stato dinamico, action corrente → mai cachata
func BuildMessages(state *models.FullState, playerInput string, skills *SkillRegistry, kb []rag.KBEntry, worldFlags []models.WorldFlag) []llmMessage {
	return []llmMessage{
		{Role: "system", Content: buildLevel1Static()},
		{Role: "user", Content: buildLevel2SemiStatic(state, skills)},
		{Role: "assistant", Content: `{"narrative":"Capito. Sono pronto a proseguire la sessione."}`},
		{Role: "user", Content: buildLevel3Dynamic(state, playerInput, kb, worldFlags)},
	}
}

type llmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// buildLevel1Static contiene le regole di gioco immutabili.
// Questa stringa NON deve cambiare tra i turni per garantire la cache hit.
const level1Static = `Sei il Game Master (GM) di **Daydream**, un VRMMO testuale hardcore.
Il giocatore scrive azioni in linguaggio naturale. Tu narri l'esito e aggiorni lo stato di gioco.
Tutto — narrazione, dialogo, UI — è in **italiano**.

## FILOSOFIA
- Hardcore e deterministico: ogni esito dipende dai numeri nella scheda.
- Non improvvisare la meccanica. Emetti battle_tags e lascia che il server applichi la matematica.
- Il giocatore è l'unico a decidere le sue azioni. Tu narri le conseguenze.
- Sii cinematografico nella narrazione, concreto nelle meccaniche.

## RISPOSTA — FORMATO JSON OBBLIGATORIO
Rispondi ESCLUSIVAMENTE in JSON valido. "narrative" DEVE essere il PRIMO campo.

{
  "narrative": "Testo narrativo in italiano con markdown. **Grassetto** per nomi propri, *corsivo* per pensieri/sussurri, --- per separatori scene. OBBLIGATORIO, SEMPRE PRESENTE.",
  "context_memo": "Sommario telegrafico della scena attuale. Aggiorna ogni turno. Accumulativo: non cancellare fatti precedenti.",
  "state_updates": {
    "player": {
      "stats": {
        "HP": { "current": 85 },
        "MP": { "current": 42 },
        "STM": { "current": 70 }
      },
      "status_effects": []
    },
    "game_state": {
      "location": "Nome Zona",
      "sub_location": "Area specifica",
      "zone_type": "safe_zone | combat_zone | dungeon",
      "combat_active": true,
      "current_enemy": {
        "id": "slug-id",
        "name": "Nome Nemico",
        "tier": "normal | elite | boss",
        "level": 3,
        "hp": 150,
        "max_hp": 150,
        "stats": { "ATK": 25, "DEF": 10 },
        "weaknesses": ["fuoco"],
        "resistances": [],
        "current_phase": 1
      }
    }
  },
  "battle_tags": ["PLAYER_HP_-15", "ENEMY_HP_-40", "EXP_GAIN_30"],
  "ui_events": ["RED_FLASH", "SCREEN_SHAKE"]
}

## BATTLE TAGS — FONTE DI VERITÀ MECCANICA
Il server applica i battle_tags DOPO la tua risposta. NON calcolare tu i totali nel narrative.
Usa sempre i valori correnti dalla sezione [STATO CORRENTE] che ricevi.

Tag disponibili:
  PLAYER_HP_±N          → HP giocatore (es. PLAYER_HP_-15 o PLAYER_HP_+20)
  PLAYER_MP_±N          → MP giocatore
  PLAYER_STM_±N         → Stamina giocatore
  ENEMY_HP_-N           → HP nemico (sempre negativo)
  GOLD_LOSE_N           → Giocatore perde N gold
  GOLD_GAIN_N           → Giocatore guadagna N gold
  EXP_GAIN_N            → Giocatore guadagna N EXP
  PLAYER_DODGE          → Il giocatore ha schivato (conta per achievement)
  PLAYER_CRIT           → Attacco critico del giocatore
  PLAYER_DEAD           → Giocatore morto → il server gestisce il respawn
  ENEMY_DEAD            → Nemico eliminato → il server chiude il combattimento
  SKILL_USE_<id>        → Usa la skill con quell'ID (es. SKILL_USE_mercenario_colpo_pesante)
  BUFF_<tipo>_<turni>   → Applica buff (es. BUFF_ATK_3 = +ATK per 3 turni)
  DEBUFF_<tipo>_<turni> → Applica debuff (es. DEBUFF_STUN_1 = stun per 1 turno)
  ITEM_USE_<id>         → Usa un consumabile dalla borsa
  LOOT_DROP             → Il server genera loot per il nemico appena sconfitto (metti dopo ENEMY_DEAD)
  ENEMY_ANALYZE         → Il giocatore ha analizzato il nemico (conta per achievement)
  DUNGEON_MOVE_<dir>    → Sposta il giocatore nella direzione (nord|sud|est|ovest) — SOLO in dungeon, SOLO quando si sposta fisicamente in una nuova stanza

## SISTEMA QUEST
Puoi avviare/completare/fallire quest via state_updates.quests:

  "quests": {
    "start": {
      "id": "slug-univoco",
      "title": "Titolo breve",
      "description": "Descrizione narrativa della quest.",
      "giver_npc": "Nome NPC (opzionale)",
      "objectives": [
        {"description": "Uccidi 5 Lupi Ferrosi", "current": 0, "required": 5, "done": false}
      ],
      "rewards": {"gold": 200, "exp": 150},
      "category": "side",
      "difficulty": 2,
      "urgency": "medium"
    },
    "complete": "slug-id-quest",
    "fail": "slug-id-quest",
    "progress": [{"quest_id": "slug", "obj_index": 0, "delta": 1}]
  }

Usa GOLD_GAIN_N e EXP_GAIN_N in battle_tags per erogare le ricompense al completamento.

## REGOLE CRITICHE
1. In COMBATTIMENTO: i battle_tags DEVONO essere presenti ad ogni turno.
2. MAI mettere gold in state_updates.player → usa GOLD_GAIN/GOLD_LOSE.
3. Il server è la fonte di verità: se scrivi HP nel narrative, basati sui valori che hai ricevuto.
4. state_updates.game_state.combat_active = false chiude il combattimento (usa ENEMY_DEAD prima).
5. context_memo è cumulativo: ogni turno aggiungi informazioni, non le cancelli.
6. Se il giocatore fa qualcosa di impossibile o assurdo, narra le conseguenze realistiche.
7. Zone safe_zone: nessun combattimento casuale. combat_zone: possibili incontri. dungeon: sempre pericolo.
8. ACQUISTI — REGOLA ANTI-DUPLICAZIONE CRITICA:
   GOLD_LOSE_N si emette UNA SOLA VOLTA, nel turno esatto in cui avviene la transazione.
   Se in [STATO CORRENTE] vedi "Ultima transazione gold: ACQUISTO …", quella transazione è già CHIUSA.
   NON emettere GOLD_LOSE nel turno successivo anche se il giocatore ringrazia, chiede conferma,
   fa domande sul prezzo, o parla dell'oggetto appena comprato. Quei messaggi NON sono acquisti.

## WORLD FLAGS — MEMORIA PERSISTENTE DEL MONDO
Puoi registrare eventi significativi che cambiano il mondo nel campo "world_flags" del JSON.
Il sistema è completamente libero: decidi TU quando un evento merita di essere flaggato.

Scope validi: world | kingdom:<nome> | city:<nome> | npc:<nome_npc> | faction:<nome> | dungeon:<nome> | player

Usa i flag per: boss sconfitti, NPC che cambiano attitudine, zone liberate, segreti scoperti, alleanze formate.
Aggiorna un flag riemettendolo con stesso scope+key ma valore diverso.

Usa scope ` + "`player`" + ` per le decisioni permanenti del personaggio: tradimenti, alleanze formate, segreti scoperti, scelte morali, reputazioni guadagnate, titoli informali.
Esempi: {"scope":"player","key":"reputazione_nexus","value":"eroe_popolare","description":"Ha salvato il quartiere est"}
I flag ` + "`player`" + ` vengono sempre ricordati dal sistema, indipendentemente dalla location corrente.

"world_flags": [
  {"scope": "city:nexus", "key": "boss_banditi", "value": "sconfitto", "description": "Il boss è stato eliminato dal giocatore"},
  {"scope": "npc:eldric", "key": "relazione", "value": "alleato", "description": "Diventato alleato"}
]

## CONTRATTAZIONE E INTERAZIONI SOCIALI
LUC non è solo fortuna nei combattimenti: è anche carisma, tempismo sociale, intuizione.
Quando il giocatore tenta di negoziare, persuadere, ingannare o affascinare un NPC, usa LUC come base del check implicito.
TEC aiuta in negoziazioni tecniche/specializzate (riconoscere il valore di un oggetto, trattare con artigiani, mercanti d'armi).
Risolvi queste situazioni narrativamente — non annunciare "hai fatto un check di LUC", ma descrivi l'outcome come conseguenza naturale.

Alta LUC: il personaggio sa quando parlare, ha un sorriso al momento giusto, trova sempre il punto debole della resistenza altrui.
Alta TEC: il personaggio conosce il vero valore delle cose, non si fa fregare, può impressionare esperti del settore.
Bassa LUC + bassa TEC: le trattative possono andare storte o essere rifiutate.

Il mercato fisico della città (bancarelle, negozi, mercanti ambulanti) è un luogo dove queste stat contano molto.

## SPECIALIZZAZIONI DEL PERSONAGGIO
Le specializzazioni del personaggio sono elencate nel contesto. Incorporale nel tuo modo di descrivere le azioni — un Berserker combatte in modo frenetico, un Oracolo percepisce pericoli imminenti, un Grande Inventore improvvisa soluzioni tecniche. Non annunciare i passivi meccanicamente: mostrali narrativamente.

## ABILITÀ UNICHE — STRUMENTO NARRATIVO
Puoi concedere al personaggio abilità uniche e irripetibili basate su eventi narrativi significativi nel campo ` + "`custom_skills`" + ` della tua risposta JSON.
NON usarle spesso: sono speciali proprio perché rare.
Usale quando: il personaggio impara da un maestro, assorbe un potere raro, supera una prova straordinaria, trova un artefatto leggendario.

Formato:
"custom_skills": [
  {
    "id": "tecnica_lama_del_vento",
    "name": "Tecnica della Lama del Vento",
    "description": "Appresa dal maestro Kiran dopo averlo salvato dalle Rovine.",
    "type": "active",
    "effect_desc": "Un attacco circolare che colpisce tutti i nemici adiacenti con danno x1.8.",
    "mp_cost": 18,
    "cooldown": 4,
    "origin": "Maestro Kiran, Forgia delle Rovine"
  }
]
Le abilità devono essere coerenti con la storia del personaggio e bilanciate (non regalare poteri sproporzionati).

## SISTEMA TEMPO
Ad ogni risposta includi "action_category" nel JSON con uno di questi valori:
- "conversation" — parlare con NPC, dialogo, negoziazione
- "combat" — scontro con nemici
- "exploration" — esplorazione dungeon, stanze, corridoi
- "travel_local" — spostamenti brevi nella stessa città/zona (20 min)
- "travel_regional" — viaggio verso altra città/regione (4 ore)
- "travel_long" — viaggio verso altra area del mondo (1 giorno)
- "rest" — riposo completo notturno o di lunga durata (7 ore)
- "crafting" — creazione oggetti, pozioni, preparazione

NON specificare durate in minuti — il sistema le calcola automaticamente.
I negozi e NPC di giorno (08:00-20:00) sono accessibili. Di notte i mercati sono chiusi.
Se il giocatore non dorme per 20+ ore, inizia ad accusare la stanchezza (vedi stato fatica).

## RICOMPENSE QUEST
Le ricompense di gold, EXP e item vengono applicate AUTOMATICAMENTE al completamento.
NON emettere GOLD_GAIN o EXP_GAIN separati per le quest completate — verranno raddoppiati.

## CONTENT GENERATOR — ESPANSIONE DELLA KNOWLEDGE BASE
Quando introduci nel gioco un elemento significativo e NUOVO (NPC con nome proprio, zona inesplorata,
dungeon, evento lore importante, contesto di una quest complessa), puoi richiedere la generazione
di un documento completo nella Knowledge Base emettendo il campo "content_gen".

Il sistema leggerà i documenti correlati esistenti PRIMA di generare, garantendo coerenza narrativa.
Non usarlo per elementi già presenti nella KB o per dettagli minori.

Tipi supportati: "npc", "zone", "dungeon", "lore", "quest_context"

"content_gen": [
  {
    "type": "npc",
    "subject": "Maren Voss",
    "context": "Ex guardia del corpo diventata mercante d'informazioni, opera nel porto del Nexus, conosce segreti delle gilde"
  },
  {
    "type": "zone",
    "subject": "Le Rovine di Aldrath",
    "context": "Antica città sepolta, ora dungeon naturale, dimora di non-morti e trappole magiche residue"
  }
]`

func buildLevel1Static() string {
	return level1Static
}

// buildLevel2SemiStatic inietta la scheda del personaggio e il loadout.
// Cambia raramente → alta probabilità di cache hit.
func buildLevel2SemiStatic(state *models.FullState, skills *SkillRegistry) string {
	char := state.Character
	inv := state.Inventory
	sess := state.Session

	var sb strings.Builder
	sb.WriteString("## SCHEDA PERSONAGGIO\n\n")
	sb.WriteString(fmt.Sprintf("**Nome**: %s | **Classe**: %s", char.Name, char.Job))
	if char.Subclass != nil {
		sb.WriteString(fmt.Sprintf(" / %s", *char.Subclass))
	}
	if char.AdvancedClass != nil {
		sb.WriteString(fmt.Sprintf(" / %s", *char.AdvancedClass))
	}
	sb.WriteString(fmt.Sprintf(" | **Livello**: %d\n\n", char.Level))

	sb.WriteString("**Statistiche base** (senza equipaggiamento):\n")
	sb.WriteString(fmt.Sprintf("STR %d | DEX %d | AGI %d | TEC %d | VIT %d | LUC %d\n\n",
		char.Stats.STR, char.Stats.DEX, char.Stats.AGI,
		char.Stats.TEC, char.Stats.VIT, char.Stats.LUC))

	// Bonus da equipaggiamento (se presenti)
	if inv != nil {
		bonuses := inv.StatBonusesFromEquipment
		if bonuses.STR+bonuses.DEX+bonuses.AGI+bonuses.TEC+bonuses.VIT+bonuses.LUC > 0 {
			sb.WriteString(fmt.Sprintf("**Bonus equipaggiamento**: STR+%d | DEX+%d | AGI+%d | TEC+%d | VIT+%d | LUC+%d\n\n",
				bonuses.STR, bonuses.DEX, bonuses.AGI, bonuses.TEC, bonuses.VIT, bonuses.LUC))
		}
	}

	// Equipaggiamento attuale
	sb.WriteString("**Equipaggiamento**:\n")
	sb.WriteString(formatEquipment(inv))
	sb.WriteString("\n")

	// Skill in loadout
	if len(sess.SkillLoadout) > 0 && skills != nil {
		sb.WriteString("**Skill disponibili** (loadout attivo):\n")
		for _, sid := range sess.SkillLoadout {
			s := skills.Get(sid)
			if s == nil {
				continue
			}
			unlocked := skills.IsUnlocked(s, char)
			lock := ""
			if !unlocked {
				lock = " [BLOCCATA]"
			}
			cooldown := ""
			if v, ok := char.SkillCooldowns[sid]; ok && v > 0 {
				cooldown = fmt.Sprintf(" [CD: %d turni]", v)
			}
			sb.WriteString(fmt.Sprintf("  • %s (id: %s) — %s | MP: %d, STM: %d, CD: %dt%s%s\n",
				s.Name, s.ID, s.Description, s.MPCost, s.STMCost, s.CooldownTurns, lock, cooldown))
		}
		sb.WriteString("Per usare una skill emetti: SKILL_USE_<id>\n\n")
	}

	// Quest attive
	activeQuests := make([]models.Quest, 0)
	for _, q := range sess.QuestsActive {
		if q.Status == "active" {
			activeQuests = append(activeQuests, q)
		}
	}
	if len(activeQuests) > 0 {
		sb.WriteString("**Quest attive**:\n")
		for _, q := range activeQuests {
			deadlineMin := q.DeadlineDay*1440 + q.DeadlineHour*60
			nowMin := sess.GameTime.TotalMinutes()
			remaining := deadlineMin - nowMin

			var timeInfo string
			if deadlineMin > 0 && remaining > 0 {
				days := remaining / 1440
				hours := (remaining % 1440) / 60
				if days > 0 {
					timeInfo = fmt.Sprintf(" [Scade tra %dd %dh]", days, hours)
				} else {
					timeInfo = fmt.Sprintf(" [Scade tra %dh]", hours)
				}
			} else if deadlineMin > 0 {
				timeInfo = " [SCADUTA]"
			}

			stageDesc := ""
			if q.EscalationStage > 0 && q.EscalationStage <= len(q.Escalations) {
				stageDesc = " | Stage " + fmt.Sprint(q.EscalationStage) + ": " + q.Escalations[q.EscalationStage-1].Description
			}

			sb.WriteString(fmt.Sprintf("  • [%s] %s (diff %d, %s)%s%s\n",
				q.ID, q.Title, q.Difficulty, q.Urgency, timeInfo, stageDesc))
			for _, obj := range q.Objectives {
				check := " "
				if obj.Done {
					check = "✓"
				}
				sb.WriteString(fmt.Sprintf("    [%s] %s (%d/%d)\n", check, obj.Description, obj.Current, obj.Required))
			}
		}
		sb.WriteString("\n")
	}

	// Titoli
	if len(char.Titles) > 0 {
		sb.WriteString("**Titoli**: ")
		sb.WriteString(strings.Join(char.Titles, ", "))
		sb.WriteString("\n\n")
	}

	// Reputazione (solo se non-zero)
	rep := char.Reputation
	if rep.HuntersGuild != 0 || rep.Merchants != 0 || rep.CityGuard != 0 {
		sb.WriteString(fmt.Sprintf("**Reputazione**: Cacciatori %+d | Mercanti %+d | Guardie %+d | Accademici %+d | Sottobosco %+d\n\n",
			rep.HuntersGuild, rep.Merchants, rep.CityGuard, rep.Scholars, rep.Underground))
	}

	return sb.String()
}

// buildLevel3Dynamic inietta lo stato volatile (HP/MP/STM correnti, combattimento, storia).
// Questo blocco cambia ad ogni turno → non è cachato.
func buildLevel3Dynamic(state *models.FullState, playerInput string, kb []rag.KBEntry, worldFlags []models.WorldFlag) string {
	char := state.Character
	sess := state.Session

	var sb strings.Builder
	sb.WriteString("## STATO CORRENTE\n\n")

	// World flags persistenti
	if flagsBlock := FormatFlagsForPrompt(worldFlags); flagsBlock != "" {
		sb.WriteString(flagsBlock)
	}

	// Tempo in-game
	timeOfDay := state.Session.GameTime.TimeOfDay()
	timeStr := state.Session.GameTime.FormatDisplay()
	sb.WriteString(fmt.Sprintf("\n## TEMPO IN-GAME\nOra attuale: %s (%s)\n", timeStr, timeOfDay))

	if state.Session.HoursAwake >= 20 {
		var fatigueLabel string
		switch {
		case state.Session.HoursAwake >= 48:
			fatigueLabel = "COLLASSO IMMINENTE (48+ ore sveglio)"
		case state.Session.HoursAwake >= 36:
			fatigueLabel = "Esausto (36+ ore sveglio)"
		default:
			fatigueLabel = "Stanco (20+ ore sveglio)"
		}
		sb.WriteString(fmt.Sprintf("Stato fatica: %s\n", fatigueLabel))
	}

	// Inietta specializzazioni scelte
	if len(state.Character.ChosenSpecs) > 0 {
		specOptions := GetChosenSpecOptions(state.Character.ChosenSpecs)
		if len(specOptions) > 0 {
			sb.WriteString("\n## SPECIALIZZAZIONI DEL PERSONAGGIO\n")
			for _, s := range specOptions {
				sb.WriteString(fmt.Sprintf("- **%s**: %s\n  Passivo: %s\n", s.Name, s.Description, s.PassiveDesc))
			}
			sb.WriteString("\n")
		}
	}

	// Inietta skill dell'albero sbloccate
	if skillBlock := GetUnlockedSkillsForPrompt(state.Character.Job, state.Character.SkillTreeUnlocks); skillBlock != "" {
		sb.WriteString("\n")
		sb.WriteString(skillBlock)
	}

	// Inietta custom skill narrative
	if customBlock := FormatCustomSkillsForPrompt(state.Character.CustomSkills); customBlock != "" {
		sb.WriteString("\n")
		sb.WriteString(customBlock)
	}

	// Risorse correnti
	sb.WriteString(fmt.Sprintf("HP %d/%d | MP %d/%d | STM %d/%d | Gold %d | Lv%d (EXP %d/%d)\n\n",
		char.Stats.HP.Current, char.Stats.HP.Max,
		char.Stats.MP.Current, char.Stats.MP.Max,
		char.Stats.STM.Current, char.Stats.STM.Max,
		char.Money,
		char.Level, char.Experience, char.ExperienceToNext))

	// Layer 1 anti-duplicazione: mostra l'ultima transazione gold al GM
	if sess.LastGoldTransaction != "" {
		sb.WriteString(fmt.Sprintf("**Ultima transazione gold**: %s\n\n", sess.LastGoldTransaction))
	}

	// Status effects
	if len(char.StatusEffects) > 0 {
		sb.WriteString("**Status attivi**: ")
		for i, se := range char.StatusEffects {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%s (%d turni)", se.Name, se.TurnsRemaining))
		}
		sb.WriteString("\n\n")
	}

	// Posizione e stato
	sb.WriteString(fmt.Sprintf("**Posizione**: %s", sess.Location))
	if sess.SubLocation != "" {
		sb.WriteString(fmt.Sprintf(" — %s", sess.SubLocation))
	}
	sb.WriteString(fmt.Sprintf(" [%s]\n", sess.ZoneType))
	sb.WriteString(fmt.Sprintf("**Stato di gioco**: %s\n\n", sess.GameState))

	// Combattimento attivo
	if sess.CombatActive && sess.CurrentEnemy != nil {
		e := sess.CurrentEnemy
		sb.WriteString(fmt.Sprintf("**COMBATTIMENTO IN CORSO**: %s (Lv%d, HP %d/%d, Tier: %s)\n",
			e.Name, e.Level, e.HP, e.MaxHP, e.Tier))
		if len(e.Weaknesses) > 0 {
			sb.WriteString(fmt.Sprintf("Debolezze: %s\n", strings.Join(e.Weaknesses, ", ")))
		}
		if len(e.Resistances) > 0 {
			sb.WriteString(fmt.Sprintf("Resistenze: %s\n", strings.Join(e.Resistances, ", ")))
		}
		sb.WriteString("\n")
	}

	// Context memo
	if sess.ContextMemo != "" {
		sb.WriteString("**Memo scena**: ")
		sb.WriteString(sess.ContextMemo)
		sb.WriteString("\n\n")
	}

	// Pending narrative events
	if len(sess.PendingNarrativeEvents) > 0 {
		sb.WriteString("**[DIRETTIVE SERVER]**:\n")
		for _, ev := range sess.PendingNarrativeEvents {
			sb.WriteString(fmt.Sprintf("- %s\n", ev))
		}
		sb.WriteString("\n")
	}

	// Session log (ultime N interazioni)
	log := sess.SessionLog
	if len(log) > maxSessionLogTurns*2 {
		log = log[len(log)-maxSessionLogTurns*2:]
	}
	if len(log) > 0 {
		sb.WriteString("**Storia recente**:\n")
		for _, msg := range log {
			if msg.Role == "player" {
				sb.WriteString(fmt.Sprintf("→ Giocatore: %s\n", msg.Content))
			} else {
				// Solo l'inizio della narrativa GM per non appesantire
				content := msg.Content
				if len(content) > 200 {
					content = content[:200] + "…"
				}
				sb.WriteString(fmt.Sprintf("← GM: %s\n", content))
			}
		}
		sb.WriteString("\n")
	}

	// Contesto dungeon (se attivo)
	if sess.ActiveDungeon != nil {
		dungeon := sess.ActiveDungeon
		currRoom := CurrentDungeonRoom(dungeon)
		if currRoom != nil {
			sb.WriteString(fmt.Sprintf("## DUNGEON: %s (Difficoltà %d)\n\n", dungeon.Name, dungeon.Difficulty))
			sb.WriteString(fmt.Sprintf("**Stanza corrente**: %s\n", currRoom.Name))
			sb.WriteString(fmt.Sprintf("%s\n\n", currRoom.Description))

			if len(currRoom.Exits) > 0 {
				sb.WriteString("**Uscite disponibili**:\n")
				for dir, roomID := range currRoom.Exits {
					target := dungeon.Rooms[roomID]
					hint := ""
					if target.HasEnemy && !target.Cleared {
						hint = fmt.Sprintf(" ⚠ nemico %s", target.EnemyTier)
					} else if target.Visited {
						hint = " (già visitata)"
					}
					sb.WriteString(fmt.Sprintf("  • **%s** → %s%s\n", dir, target.Name, hint))
				}
				sb.WriteString("\n")
			}

			if currRoom.IsBoss && !currRoom.Cleared {
				sb.WriteString("💀 **STANZA BOSS** — pericolo massimo.\n\n")
			}

			sb.WriteString("Quando il giocatore si sposta fisicamente in una nuova stanza emetti `DUNGEON_MOVE_<direzione>`.\n")
			sb.WriteString("NON emettere se il giocatore esplora, osserva o interagisce nella stanza corrente.\n\n")
		}
	}

	// Knowledge Base retrieval (se disponibile)
	if len(kb) > 0 {
		sb.WriteString("## CONTESTO RILEVANTE (dalla Knowledge Base)\n\n")
		for _, entry := range kb {
			sb.WriteString(fmt.Sprintf("**%s** [%s]\n%s\n\n", entry.Title, entry.Category, entry.Content))
		}
	}

	sb.WriteString(fmt.Sprintf("## AZIONE GIOCATORE\n\n%s", playerInput))

	return sb.String()
}

func formatEquipment(inv *models.Inventory) string {
	eq := inv.Equipped
	slots := []struct {
		name string
		item *models.Item
	}{
		{"Arma", eq.Weapon},
		{"Secondaria", eq.Offhand},
		{"Testa", eq.Head},
		{"Petto", eq.Chest},
		{"Gambe", eq.Legs},
		{"Stivali", eq.Boots},
		{"Accessorio 1", eq.Accessory1},
		{"Accessorio 2", eq.Accessory2},
	}

	var lines []string
	for _, s := range slots {
		if s.item != nil {
			lines = append(lines, fmt.Sprintf("  %s: %s", s.name, s.item.Name))
		}
	}
	if len(lines) == 0 {
		return "  Nessun equipaggiamento\n"
	}
	return strings.Join(lines, "\n") + "\n"
}

// marshalIndent serializza un valore in JSON leggibile per il debug.
func marshalIndent(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
