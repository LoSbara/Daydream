# Daydream — Architettura Tecnica e Piano Implementativo

> Documento di design: analisi critica della specifica + architettura finale + roadmap.  
> Basato su: spec originale "Overflow/Aether Horizon", esperienza dal prototipo Shanfro/server.js (~1900 righe).  
> Data: 2026-06-17 | Aggiornato: 2026-06-19
1
---

## Indice

1. [Analisi Critica della Specifica](#1-analisi-critica)
2. [Scelte Tecnologiche](#2-scelte-tecnologiche)
3. [Architettura Finale](#3-architettura-finale)
4. [SurrealDB — Schema e RAG](#4-surrealdb)
5. [Backend Go — Struttura e Pipeline](#5-backend-go)
6. [Agent System e Pipelines](#6-agent-system)
7. [MCP Tools](#7-mcp-tools)
8. [Frontend React](#8-frontend-react)
9. [Roadmap Implementativa](#9-roadmap)

---

## 1. Analisi Critica

### 1.1 Punti Forti della Specifica (da conservare)

- **Battle Tags Engine**: eccellente separazione tra narrazione LLM e meccanica server-side. Il GM emette tag semantici, il server è fonte di verità matematica. Questo pattern elimina gli errori di calcolo dell'AI ed è la cosa più intelligente del design.
- **3-livello memory (verbatim / context_memo / diary)**: gestione pragmatica del context window. Il memo accumulativo per la scena è il pattern giusto.
- **Server-authoritative money**: blocco pre-flight prima di `bag_add` con ledger transazioni. Corretto.
- **Phase triggers per boss**: separazione dati (soglie HP) da logica (stat modifiers). Estendibile.
- **Streaming SSE dual-phase**: token narrativi in real-time + payload JSON finale. Provato e funzionante in Shanfro.

### 1.2 Punti Deboli Critici

#### A. Context Memo Illimitato
La regola "context_memo deve essere accumulativo, non cancellare fatti pregressi" porta a un memo che cresce senza limite nel corso di una sessione lunga. Con sessioni di 50+ turni, il memo supera i 2000 token e occupa metà del context window del GM, riducendo lo spazio per history e stato personaggio.

**Fix**: introdurre un `ContextCompactor Agent` che ogni N turni (o quando il memo supera una soglia token) genera un sommario compresso. Il nuovo memo sostituisce il vecchio; le informazioni importanti vengono promosse al diary. Trigger: `len(context_memo) > 800 token`.

#### B. Il GM come Unico Agente
La specifica descrive un sistema multi-agente ("diversi agenti AI") ma poi tratta il GM come unica entità che fa tutto: narrare, aggiornare stato, gestire shop, gestire NPC, generare dungeon lore. Questo crea prompt mostruosi e alta probabilità di allucinazioni su compiti secondari.

**Fix**: specializzare gli agenti per ruolo (vedi sezione 6). Il GM Agent si occupa solo di narrazione e battle tags; agenti separati gestiscono la generazione di contenuti, la validazione e la compressione del contesto.

#### C. Nessun Meccanismo di Retry su JSON Invalido
La spec non menziona cosa succede se il GM restituisce JSON malformato. Nel prototipo Shanfro questo è il crash più comune: una parentesi mancante azzera il turno. Con un solo agente GM non c'è fallback.

**Fix**: pipeline di parsing con retry. Se il parse fallisce: (1) richiama il GM con il buffer e istruzione "correggi il JSON", max 2 retry; (2) se fallisce ancora, torna al server con stato invariato + messaggio di errore narrativo generico ("Il tuo GM si è momentaneamente disconnesso...").

#### D. Race Condition su `pending_narrative_events`
Gli eventi iniettati da pre-compute (trappole, respawn, boss phase) vengono aggiunti all'array e consumati al turno successivo. Se il client invia due messaggi prima che il server abbia completato il turno precedente, la coda degli eventi si mescola con il turno sbagliato.

**Fix**: la FIFO queue per-utente del prototipo Shanfro (già implementata con promise chaining) deve diventare esplicita con goroutine channel in Go. `pending_narrative_events` va versioned con il `turn_id` per associazione sicura.

#### E. RAG su SurrealDB: L'Embedding è Esterno
SurrealDB supporta vector search (HNSW) ma non genera embeddings. La specifica non menziona da dove vengono gli embedding, chi li genera, e quando vengono aggiornati. Senza questa pipeline il RAG non funziona.

**Fix**: definire la pipeline di embedding (sezione 4.3). Endpoint dedicato per ingestion, embedding generato via API esterna (Jina, OpenAI text-embedding-3-small) o modello locale compatibile. Il processo è asincrono rispetto al turno di gioco.

#### F. Stato del Gioco è Flat, Non una State Machine
Lo stato del gioco ha campi `combat_active`, `zone_type`, `current_dungeon_id` che si contraddicono se non aggiornati in modo coordinato. Es: `combat_active=true` e `zone_type='safe_zone'` è uno stato invalido ma non c'è nulla che lo prevenga.

**Fix**: modellare lo stato come macchina a stati esplicita con transizioni validate: `WORLD_NAVIGATION → COMBAT → WORLD_NAVIGATION`, `WORLD_NAVIGATION → DUNGEON_EXPLORATION → DUNGEON_COMBAT → DUNGEON_EXPLORATION → WORLD_NAVIGATION`, ecc. Ogni transizione è un metodo Go che valida il contesto prima di applicare il cambio.

#### G. Nessuna Autenticazione nella Specifica
Il sistema multi-utente è citato ma non c'è menzione di login, JWT, sessioni. Il prototipo usa `X-User-Id` come header non autenticato — chiunque può impersonare un altro utente passando un id diverso.

**Fix**: JWT auth obbligatorio. Login con username/password (bcrypt), refresh token, middleware auth su tutti gli endpoint protetti.

#### H. Global Announcements senza Broadcast Channel
La spec cita "annunci globali visibili a tutti i player" ma il backend HTTP con SSE per-utente non ha un meccanismo di broadcast nativo. Ogni SSE è una connessione isolata per utente.

**Fix**: un hub WebSocket separato (goroutine condivisa) che mantiene tutte le connessioni attive e fa broadcast sugli eventi globali. SSE per il turno di gioco (unidirezionale, più semplice), WebSocket per gli annunci globali (broadcast).

#### I. Il Dashboard Builder Richiede Architettura Dedicata
"Altamente modulare e componibile in base alle esigenze" è una feature non banale. Non è solo un layout diverso per ogni campagna: richiede un sistema di panel registration, persist del layout per utente, e componenti autonomi che non si rompono se spostati.

**Fix**: Panel System con `react-grid-layout`. Ogni panel è un componente isolato con prop `state` iniettato dall'esterno. Il layout è un array di `{i, x, y, w, h}` persistito in SurrealDB per utente. La config del dashboard è part of campaign/character data.

### 1.3 Lacune della Specifica (da aggiungere)

| Lacuna | Impatto | Soluzione |
|--------|---------|-----------|
| Rate limiting per endpoint | Un client malevolo può spammare il GM AI | Token bucket per-utente, max 1 req/2s per `/api/chat` |
| Validazione input utente | Injection nel system prompt | Sanitize + max length (500 chars) su ogni messaggio player |
| Backup asincrono | File JSON corrotti in Shanfro sono il bug più frequente | SurrealDB transactions + undo log; export periodico in S3 |
| Context window dinamico | Diversi provider hanno limiti diversi | `MAX_CONTEXT_TOKENS` configurabile per provider; compaction automatica se si avvicina al limite |
| Metriche di qualità narrazione | Come verificare che il GM sia "buono"? | Validator Agent con scoring su JSON completeness, battle tag accuracy, narrative length |
| Dungeon procedurale | Spec dice "generazione room-by-room" ma non come | Dungeon Generator Agent che espande template statici (sezione 6.5) |
| Seed deterministico per drop/critico | `Math.random()` non è riproducibile in Go | Usa `rand.New(rand.NewSource(seed))` con seed derivato da `(turn_id XOR player_id_hash)` |

---

## 2. Scelte Tecnologiche

### 2.1 Backend: Go

**Raccomandazione: Go (Golang) con Gin**

Confronto rapido:

| Criterio | Node.js | Go | Rust |
|----------|---------|-----|------|
| Concorrenza multi-utente | Buona (event loop) | **Eccellente (goroutines)** | Eccellente (Tokio) |
| SSE Streaming | Ottima (streams native) | Ottima (net/http) | Buona (axum) |
| Tempo di sviluppo | Rapido | **Rapido-Medio** | Lento |
| Maturità SurrealDB SDK | surrealdb.js (buono) | **surrealdb.go (stabile)** | surrealdb (in sviluppo) |
| Gestione code FIFO per-utente | Promise chaining (fragile) | **Channel + goroutine (elegante)** | Tokio mpsc |
| LLM-agnostic HTTP client | fetch (built-in) | **net/http (built-in)** | reqwest |
| JSON structured output | Ottimo | **Ottimo** | Ottimo |

**Motivazione per Go**: il problema principale di Shanfro era la gestione della concorrenza (FIFO queue implementata con promise chaining in JavaScript, fragile). In Go ogni player ha una goroutine con un `channel` dedicato — è il pattern nativo del linguaggio. Il codice di concorrenza è più leggibile e più sicuro. Gin è lo standard de-facto per REST API in Go con eccellente middleware ecosystem.

### 2.2 Database: SurrealDB

Versione: **SurrealDB v2.x** (supporta HNSW vector search)

Utilizzato come:
- DB principale per tutto lo stato di gioco (sostituisce i file JSON)
- Vector store per il RAG (HNSW index su campo `embedding`)
- Document store per i cataloghi statici

Non serve un database vettoriale separato (Qdrant, Pinecone) perché SurrealDB v2 supporta HNSW nativamente con performance adeguate per questo use case (<1M documenti).

### 2.3 Frontend: React + Vite + JavaScript

Stack UI:
- **React 18** + **Vite** (HMR veloce, build ottimizzata)
- **shadcn/ui** + **Tailwind CSS** (già presente in Shanfro "Logo" folder — componenti pronti)
- **Zustand** per state management (leggero, senza boilerplate Redux)
- **react-grid-layout** per il dashboard builder
- **react-markdown** per il rendering della narrativa GM
- **Recharts** per grafici stat/exp (opzionale)
- Niente TypeScript — JavaScript puro come richiesto

### 2.4 Embedding e LLM

**LLM**: adapter layer LLM-agnostic con interfaccia comune. Provider configurabili:
- OpenAI-compatible (OpenAI, DeepSeek, Groq, Ollama)
- Anthropic Claude API
- Google Gemini API

**Embedding**: `nomic-embed-text` via Ollama per uso locale (768 dim) **oppure** `text-embedding-3-small` OpenAI (1536 dim) per cloud. Selezionabile da config.

### 2.5 Comunicazione Real-time

- **SSE** (`text/event-stream`): turni di gioco (narrativa streaming + payload finale). Connessione per-utente, unidirezionale.
- **WebSocket**: global announcements, notifiche eventi eccezionali visibili a tutti i player connessi.

---

## 3. Architettura Finale

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                            CLIENT (React + Vite)                                │
│                                                                                 │
│  ┌──────────────┐  ┌──────────────────────────────────┐  ┌──────────────────┐   │
│  │  Dashboard   │  │         Chat Panel               │  │   Stats/Inv/     │   │
│  │  Builder     │  │  (markdown streaming + history)  │  │   Map Panels     │   │
│  │  (Grid       │  │                                  │  │  (panel registry)│   │
│  │  Layout)     │  │    EventSource (SSE)             │  │                  │   │
│  └──────────────┘  └──────────────────────────────────┘  └──────────────────┘   │
│                              │  WebSocket (global annunci)                      │
└──────────────────────────────┼──────────────────────────────────────────────────┘
                               │
                    ┌──────────▼──────────┐
                    │   Go Backend (Gin)  │
                    │                     │
                    │  ┌───────────────┐  │
                    │  │  Auth (JWT)   │  │
                    │  └───────────────┘  │
                    │  ┌───────────────┐  │
                    │  │  Turn Engine  │  │     ┌─────────────────────┐
                    │  │  (state mach.)│──┼────►│  LLM Adapter Layer  │
                    │  └───────────────┘  │     │  (OpenAI/Anthropic/ │
                    │  ┌───────────────┐  │     │   Gemini/Ollama)    │
                    │  │  Battle Tags  │  │     └─────────────────────┘
                    │  │  Engine       │  │
                    │  └───────────────┘  │     ┌─────────────────────┐
                    │  ┌───────────────┐  │     │  Agent Orchestrator │
                    │  │  Agent Proxy  │──┼────►│  (GM / Validator /  │
                    │  └───────────────┘  │     │  Compactor /        │
                    │  ┌───────────────┐  │     │  ContentGen /       │
                    │  │  WS Broadcast │  │     │  DungeonGen)        │
                    │  │  Hub          │  │     └─────────────────────┘
                    │  └───────────────┘  │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │    SurrealDB v2     │
                    │                     │
                    │  ┌───────────────┐  │     ┌─────────────────────┐
                    │  │  Game State   │  │     │  Embedding Service  │
                    │  │  (per-user)   │  │     │  (nomic/openai)     │
                    │  └───────────────┘  │◄────┤  Pipeline async     │
                    │  ┌───────────────┐  │     └─────────────────────┘
                    │  │  Knowledge    │  │
                    │  │  Base (HNSW   │  │     ┌─────────────────────┐
                    │  │  vector idx)  │  │     │  MCP Tools Server   │
                    │  └───────────────┘  │     │  (Go, stdio/HTTP)   │
                    │  ┌───────────────┐  │     └─────────────────────┘
                    │  │  Cataloghi    │  │
                    │  │  statici      │  │
                    │  └───────────────┘  │
                    └─────────────────────┘
```

### 3.1 Flusso Turno di Gioco

```
Player invia messaggio
        │
        ▼
[1] Auth middleware (JWT validation)
        │
        ▼
[2] Rate limiter (token bucket per-utente, max 1 req/2s)
        │
        ▼
[3] Input sanitizer (max 500 chars, rimuovi injection patterns)
        │
        ▼
[4] Enqueue su goroutine channel per-utente (FIFO serialization)
        │  ← goroutine player-specific prende il controllo
        ▼
[5] Load state (SurrealDB transaction BEGIN)
        ├── character, inventory, game_session, skills, npcs
        └── pending_narrative_events (consumed + cleared)
        │
        ▼
[6] Pre-compute server-side (PRIMA della chiamata LLM)
        ├── tickStatusEffects (turno effetti, STATUS_ADD/REMOVE)
        ├── tickSkillCooldowns
        ├── dungeonPrecompute (trappole AGI check, puzzle TEC check)
        ├── consumePendingEvents (inietta in serverDirectives)
        └── validateGameState (state machine check)
        │
        ▼
[7] RAG Retrieval (async, max 100ms timeout)
        ├── embed(player_message + location + context)
        ├── SELECT top-5 FROM knowledge_base (HNSW search)
        └── inject relevant_lore_chunks into context
        │
        ▼
[8] Build GM Prompt (3 livelli per prefix cache)
        ├── Livello 1 STATICO: regole gioco, schema JSON risposta, lore assiomi
        ├── Livello 2 SEMI-STATICO: personaggio, classi, skill, equipaggiamento,
        │   NPC zona corrente, world flags, zone lore, RAG chunks
        └── Livello 3 DINAMICO: HP/MP/STM correnti, combat_state,
            context_memo, session_log ultimi 12, serverDirectives
        │
        ▼
[9] GM Agent LLM Call (streaming SSE al client)
        ├── SSE emit: data: {"type":"token","text":"..."} per ogni token narrativo
        ├── Retry su JSON parse fail (max 2 retry con istruzione "correggi JSON")
        └── SSE emit: {"type":"parse_error"} se tutti i retry falliscono
        │
        ▼
[10] Parse + Validate GM Response
        ├── Validator Agent (async, non blocca il turno): verifica battle tag plausibility,
        │   narrative length, JSON completeness → score 0-100 loggato per osservabilità
        └── Filtra campi vietati (money in state_updates → silently ignored)
        │
        ▼
[11] Apply State Updates (SurrealDB transaction)
        ├── processBattleTags (fonte di verità matematica)
        │   ├── COMBAT_HIT_PLAYER: calcola danno da stats reali, check game_over
        │   ├── COMBAT_HIT_ENEMY: danno a nemico, part break check, phase trigger check
        │   ├── GOLD_LOSE_N: pre-flight check fondi, ledger entry
        │   ├── GOLD_GAIN_N: ledger entry
        │   ├── SKILL_USE_<id>: verifica MP/STM/CD, applica effetto, scala cooldown
        │   └── EXP_GAIN_N, BAG_ADD/REMOVE, STATUS_ADD/REMOVE, ...
        ├── deepMerge(state_updates.player)
        ├── deepMerge(state_updates.game_state)
        ├── applyBagAdd / appraise_item / reputation_delta
        ├── npc_add / npc_update
        ├── context_memo update (+ compaction check → trigger async ContextCompactor)
        └── diary_entry persist
        │
        ▼
[12] Post-compute
        ├── checkLevelUp (auto level-up con reward)
        ├── checkSkillUnlocks
        ├── checkTitles
        ├── checkQuestProgress
        └── checkGameOver (HP ≤ 0 → respawn, -20% oro, teleport Crysta)
        │
        ▼
[13] Commit SurrealDB transaction
        │
        ▼
[14] SSE emit: data: {"type":"done", payload: fullStatePayload}
        │
        ▼
[15] Async post-turn (non bloccante)
        ├── Embedding pipeline (se nuovo diary entry o NPC)
        ├── Global announcement (se boss kill, item leggendario valutato)
        └── Auto-backup trigger (se level_up o unique event)
```

---

## 4. SurrealDB — Schema e RAG

### 4.1 Schema Tabelle

```surql
-- ============================================================
-- AUTENTICAZIONE
-- ============================================================
DEFINE TABLE user SCHEMAFULL;
DEFINE FIELD username     ON user TYPE string  ASSERT string::len($value) >= 3;
DEFINE FIELD password_hash ON user TYPE string;
DEFINE FIELD created_at   ON user TYPE datetime;
DEFINE FIELD last_login   ON user TYPE datetime;
DEFINE INDEX idx_username ON user FIELDS username UNIQUE;

-- ============================================================
-- PERSONAGGIO E STATO DI GIOCO (per-utente, mutable)
-- ============================================================
DEFINE TABLE character SCHEMAFULL;
DEFINE FIELD user_id        ON character TYPE record<user>;
DEFINE FIELD name           ON character TYPE string;
DEFINE FIELD job            ON character TYPE string;  -- classe T0
DEFINE FIELD subclass       ON character TYPE option<string>;
DEFINE FIELD advanced_class ON character TYPE option<string>;
DEFINE FIELD level          ON character TYPE int    DEFAULT 1;
DEFINE FIELD experience     ON character TYPE int    DEFAULT 0;
DEFINE FIELD experience_to_next ON character TYPE int DEFAULT 100;
DEFINE FIELD stats          ON character TYPE object;
  -- { HP: {current, max}, MP: {current, max}, STM: {current, max},
  --   STR, DEX, AGI, TEC, VIT, LUC }
DEFINE FIELD stat_points_available ON character TYPE int DEFAULT 0;
DEFINE FIELD money          ON character TYPE int DEFAULT 500;
DEFINE FIELD skill_slots    ON character TYPE int DEFAULT 4;
DEFINE FIELD titles         ON character TYPE array DEFAULT [];
DEFINE FIELD status_effects ON character TYPE array DEFAULT [];
DEFINE FIELD reputation     ON character TYPE object;
DEFINE FIELD flags          ON character TYPE object DEFAULT {};
DEFINE FIELD action_counters ON character TYPE object;
DEFINE FIELD skill_cooldowns ON character TYPE object DEFAULT {};
DEFINE INDEX idx_char_user ON character FIELDS user_id;

DEFINE TABLE inventory SCHEMAFULL;
DEFINE FIELD character_id ON inventory TYPE record<character>;
DEFINE FIELD equipped      ON inventory TYPE object;
DEFINE FIELD stat_bonuses_from_equipment ON inventory TYPE object;
DEFINE FIELD bag           ON inventory TYPE array DEFAULT [];
DEFINE INDEX idx_inv_char ON inventory FIELDS character_id UNIQUE;

DEFINE TABLE game_session SCHEMAFULL;
DEFINE FIELD character_id        ON game_session TYPE record<character>;
DEFINE FIELD location            ON game_session TYPE string DEFAULT "Crysta";
DEFINE FIELD sub_location        ON game_session TYPE string DEFAULT "";
DEFINE FIELD zone_type           ON game_session TYPE string DEFAULT "safe_zone";
DEFINE FIELD game_state          ON game_session TYPE string DEFAULT "WORLD_NAVIGATION";
  -- Enum: MENU | WORLD_NAVIGATION | COMBAT | DUNGEON_EXPLORATION | DUNGEON_COMBAT
DEFINE FIELD combat_active       ON game_session TYPE bool DEFAULT false;
DEFINE FIELD current_enemy       ON game_session TYPE option<object>;
DEFINE FIELD tactical_tension    ON game_session TYPE int DEFAULT 0;
DEFINE FIELD skill_loadout       ON game_session TYPE array DEFAULT [];
DEFINE FIELD session_log         ON game_session TYPE array DEFAULT [];  -- rolling 12
DEFINE FIELD context_memo        ON game_session TYPE string DEFAULT "";
DEFINE FIELD context_memo_tokens ON game_session TYPE int DEFAULT 0;  -- stimato
DEFINE FIELD current_dungeon_id  ON game_session TYPE option<string>;
DEFINE FIELD current_room_id     ON game_session TYPE option<string>;
DEFINE FIELD rooms_visited       ON game_session TYPE array DEFAULT [];
DEFINE FIELD pending_narrative_events ON game_session TYPE array DEFAULT [];
DEFINE FIELD quests_active       ON game_session TYPE array DEFAULT [];
DEFINE FIELD quests_completed    ON game_session TYPE array DEFAULT [];
DEFINE FIELD unique_scenario_flags ON game_session TYPE object DEFAULT {};
DEFINE FIELD party               ON game_session TYPE array DEFAULT [];
DEFINE FIELD threat_table        ON game_session TYPE object DEFAULT {};
DEFINE FIELD turn_id             ON game_session TYPE int DEFAULT 0;
DEFINE FIELD dashboard_layout    ON game_session TYPE option<array>;
  -- react-grid-layout config: [{i, x, y, w, h}]
DEFINE INDEX idx_session_char ON game_session FIELDS character_id UNIQUE;

-- NPC per-utente (relazioni e stato)
DEFINE TABLE npc_instance SCHEMAFULL;
DEFINE FIELD character_id  ON npc_instance TYPE record<character>;
DEFINE FIELD npc_id        ON npc_instance TYPE string;  -- ref a npc_catalog
DEFINE FIELD name          ON npc_instance TYPE string;
DEFINE FIELD faction       ON npc_instance TYPE string;
DEFINE FIELD relationship  ON npc_instance TYPE int DEFAULT 0;
DEFINE FIELD notes         ON npc_instance TYPE string DEFAULT "";
DEFINE FIELD last_seen     ON npc_instance TYPE option<string>;
DEFINE FIELD location      ON npc_instance TYPE option<string>;
DEFINE INDEX idx_npc_char ON npc_instance FIELDS character_id, npc_id UNIQUE;

-- Bestiario per-utente
DEFINE TABLE bestiary_entry SCHEMAFULL;
DEFINE FIELD character_id ON bestiary_entry TYPE record<character>;
DEFINE FIELD monster_id   ON bestiary_entry TYPE string;
DEFINE FIELD name         ON bestiary_entry TYPE string;
DEFINE FIELD tier         ON bestiary_entry TYPE string;
DEFINE FIELD weaknesses   ON bestiary_entry TYPE array DEFAULT [];
DEFINE FIELD parts_broken ON bestiary_entry TYPE array DEFAULT [];
DEFINE INDEX idx_bestiary ON bestiary_entry FIELDS character_id, monster_id UNIQUE;

-- Diario di viaggio per-utente
DEFINE TABLE travel_diary SCHEMAFULL;
DEFINE FIELD character_id ON travel_diary TYPE record<character>;
DEFINE FIELD turn_id      ON travel_diary TYPE int;
DEFINE FIELD location     ON travel_diary TYPE string;
DEFINE FIELD sub_location ON travel_diary TYPE string;
DEFINE FIELD summary      ON travel_diary TYPE string;
DEFINE FIELD npcs         ON travel_diary TYPE array DEFAULT [];
DEFINE FIELD created_at   ON travel_diary TYPE datetime;
DEFINE INDEX idx_diary_char ON travel_diary FIELDS character_id;

-- Ledger transazioni denaro (immutabile, append-only)
DEFINE TABLE transaction_ledger SCHEMAFULL;
DEFINE FIELD character_id ON transaction_ledger TYPE record<character>;
DEFINE FIELD turn_id      ON transaction_ledger TYPE int;
DEFINE FIELD type         ON transaction_ledger TYPE string;  -- GOLD_GAIN | GOLD_LOSE
DEFINE FIELD amount       ON transaction_ledger TYPE int;
DEFINE FIELD balance_after ON transaction_ledger TYPE int;
DEFINE FIELD reason       ON transaction_ledger TYPE string;  -- "battle_tag:GOLD_LOSE_50"
DEFINE FIELD created_at   ON transaction_ledger TYPE datetime;
DEFINE INDEX idx_ledger_char ON transaction_ledger FIELDS character_id;

-- Slot di salvataggio
DEFINE TABLE save_slot SCHEMAFULL;
DEFINE FIELD character_id ON save_slot TYPE record<character>;
DEFINE FIELD slot_id      ON save_slot TYPE int;  -- 1, 2, 3
DEFINE FIELD snapshot     ON save_slot TYPE object;  -- JSON snapshot completo
DEFINE FIELD saved_at     ON save_slot TYPE datetime;
DEFINE INDEX idx_slot ON save_slot FIELDS character_id, slot_id UNIQUE;

-- ============================================================
-- CATALOGHI STATICI (condivisi tra tutti i player)
-- ============================================================
DEFINE TABLE skill_catalog     SCHEMAFULL;  -- 249 skills
DEFINE TABLE class_catalog     SCHEMAFULL;  -- 5 T0 + 16 T2 + 32 T3
DEFINE TABLE monster_catalog   SCHEMAFULL;  -- con drop table, parts, phase_triggers
DEFINE TABLE item_catalog      SCHEMAFULL;  -- oggetti unici, leggendari
DEFINE TABLE quest_catalog     SCHEMAFULL;  -- quest con objectives e rewards
DEFINE TABLE dungeon_template  SCHEMAFULL;  -- template dungeon (stanze, tipi, connessioni)
DEFINE TABLE recipe_catalog    SCHEMAFULL;  -- ricette crafting
DEFINE TABLE title_catalog     SCHEMAFULL;  -- 15 titoli con condizioni e rewards
DEFINE TABLE world_zone        SCHEMAFULL;  -- zone mappa con lore, connections, zone_type
DEFINE TABLE shop_catalog      SCHEMAFULL;  -- assortimento negozio per zona

-- Annunci globali
DEFINE TABLE global_announcement SCHEMAFULL;
DEFINE FIELD type       ON global_announcement TYPE string;
DEFINE FIELD message    ON global_announcement TYPE string;
DEFINE FIELD player     ON global_announcement TYPE string;
DEFINE FIELD created_at ON global_announcement TYPE datetime;

-- ============================================================
-- KNOWLEDGE BASE (RAG con vector search)
-- ============================================================
DEFINE TABLE knowledge_base SCHEMAFULL;
DEFINE FIELD doc_id     ON knowledge_base TYPE string;
DEFINE FIELD type       ON knowledge_base TYPE string;
  -- zone_lore | npc_personality | world_axiom | quest_lore | item_lore
  -- | diary_summary | session_summary | faction_lore
DEFINE FIELD scope      ON knowledge_base TYPE string;
  -- global | character:<id>  (global = tutti, character = solo quel player)
DEFINE FIELD text       ON knowledge_base TYPE string;  -- chunk originale
DEFINE FIELD embedding  ON knowledge_base TYPE array;   -- vettore float[]
DEFINE FIELD metadata   ON knowledge_base TYPE object;  -- {source, zone, tags}
DEFINE FIELD created_at ON knowledge_base TYPE datetime;
DEFINE FIELD updated_at ON knowledge_base TYPE datetime;

-- Indice HNSW per vector search
DEFINE INDEX idx_kb_embedding ON knowledge_base
  FIELDS embedding HNSW DIMENSION 768 DIST COSINE;
-- Nota: usa 768 per nomic-embed-text (locale); cambia a 1536 per text-embedding-3-small (OpenAI)

-- Indice testuale per keyword search
DEFINE ANALYZER knowledge_analyzer TOKENIZERS blank, class, camel
  FILTERS lowercase, snowball(italian);
DEFINE INDEX idx_kb_text ON knowledge_base
  FIELDS text SEARCH ANALYZER knowledge_analyzer BM25;
```

### 4.2 Query RAG nel Turn Engine

```surql
-- Hybrid search: vector similarity + keyword BM25
-- Chiamata al momento del step [7] del flusso turno

LET $query_vec = $embedding_of_player_message_plus_context;
LET $player_char = $character_id;

SELECT *, vector::similarity::cosine(embedding, $query_vec) AS score
FROM knowledge_base
WHERE scope = 'global' OR scope = $player_char
  AND (
    embedding <|5, 40|> $query_vec  -- top-5 HNSW (ef=40)
    OR text @@ $player_message       -- BM25 fulltext
  )
ORDER BY score DESC
LIMIT 8
FETCH metadata;
```

La query è ibrida: HNSW per similarità semantica, BM25 per keyword esatte (nomi propri, id oggetti). Top-8 chunk vengono iniettati nel Livello 2 del prompt (~400 token riservati).

### 4.3 Embedding Pipeline (asincrona)

```
Trigger → Embedding Worker (goroutine) → SurrealDB

Trigger events:
  - Nuovo diary_entry creato (per personaggio specifico)
  - Nuovo NPC creato o aggiornato con note significative
  - Boss sconfitto → inserisce "axiom" nel knowledge_base
  - Admin carica nuovo contenuto lore (zona, quest)
  - Context Compactor Agent produce un riassunto sessione

Worker steps:
  1. Ricevi documento da channel
  2. Chunk il testo (512 token, 50 token overlap, boundary su sentence)
  3. Per ogni chunk: POST /embeddings → embedding model → vettore float[]
  4. UPSERT INTO knowledge_base (doc_id = hash(testo), ...)
  5. Log completion
```

Il worker è una goroutine separata che consuma una `chan EmbeddingTask`. Non blocca mai il turn engine.

---

## 5. Backend Go — Struttura e Pipeline

### 5.1 Struttura Directory

```
aether-horizon/
├── cmd/
│   ├── server/
│   │   └── main.go           # entry point, setup Gin, migrate DB, start workers
│   └── seed/
│       └── main.go           # CLI per caricare cataloghi statici in SurrealDB
├── internal/
│   ├── auth/
│   │   ├── jwt.go            # generate/validate JWT
│   │   └── middleware.go     # Gin middleware auth
│   ├── db/
│   │   ├── client.go         # SurrealDB connection pool
│   │   ├── migrate.go        # schema migrations
│   │   └── queries/          # SurrealQL queries come string constants
│   ├── game/
│   │   ├── engine.go         # orchestratore turno (i 15 step del flusso)
│   │   ├── statemachine.go   # game state machine con transizioni validate
│   │   ├── battletagsengine.go # processBattleTags() — fonte di verità meccanica
│   │   ├── formulas.go       # danno, schivata, critico, analisi
│   │   ├── levelup.go        # checkLevelUp, checkSkillUnlocks, checkTitles
│   │   ├── questengine.go    # checkQuestProgress
│   │   ├── gameover.go       # respawn logic
│   │   └── inventory.go      # equip, unequip, craft, enhance, appraise
│   ├── llm/
│   │   ├── adapter.go        # interfaccia LLMProvider {Chat, Stream, Embed}
│   │   ├── openai.go         # provider OpenAI-compatible
│   │   ├── anthropic.go      # provider Anthropic
│   │   ├── gemini.go         # provider Google Gemini
│   │   └── config.go         # selezione provider da env
│   ├── agents/
│   │   ├── gm.go             # GM Agent: build prompt, call LLM, parse response
│   │   ├── validator.go      # Validator Agent: score risposta GM
│   │   ├── compactor.go      # Context Compactor Agent: comprime context_memo
│   │   ├── contentgen.go     # Content Generator Agent: NPC/dungeon lore
│   │   └── dungeongen.go     # Dungeon Generator Agent: espande template
│   ├── rag/
│   │   ├── retriever.go      # hybrid search su SurrealDB
│   │   ├── embedder.go       # interfaccia embedding + implementazioni
│   │   └── pipeline.go       # goroutine worker per embedding asincrono
│   ├── queue/
│   │   └── playerqueue.go    # FIFO per-player con goroutine + channel
│   ├── broadcast/
│   │   └── hub.go            # WebSocket broadcast hub per global announcements
│   ├── api/
│   │   ├── router.go         # setup Gin routes
│   │   ├── chat.go           # POST /api/chat (SSE streaming)
│   │   ├── state.go          # GET /api/state
│   │   ├── character.go      # allocate, subclass, advanced-class, reset
│   │   ├── inventory.go      # equip, unequip, use-item, enhance, appraise
│   │   ├── shop.go           # shop endpoints
│   │   ├── quests.go         # quests endpoints
│   │   ├── dungeon.go        # dungeon endpoints
│   │   ├── slots.go          # save/load slots
│   │   ├── catalog.go        # world-map, bestiary, npcs, diary, recipes
│   │   └── admin.go          # endpoints GM mode, seed knowledge base
│   └── middleware/
│       ├── ratelimit.go      # token bucket per-user
│       └── sanitize.go       # input sanitization
├── configs/
│   ├── schema.surql          # schema SurrealDB (versioned)
│   └── seed/                 # JSON cataloghi statici per il seeder
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

### 5.2 Interfaccia LLM-Agnostic

```go
// internal/llm/adapter.go

type ChatMessage struct {
    Role    string `json:"role"` // "system" | "user" | "assistant"
    Content string `json:"content"`
}

type StreamToken struct {
    Text    string
    Done    bool
    UsageIn  int // per logging token
    UsageOut int
}

type LLMProvider interface {
    Chat(ctx context.Context, messages []ChatMessage, schema *JSONSchema) (string, error)
    Stream(ctx context.Context, messages []ChatMessage, schema *JSONSchema, out chan<- StreamToken) error
    Embed(ctx context.Context, text string) ([]float32, error)
    Name() string
}

// Selezione provider da env: LLM_PROVIDER=openai|anthropic|gemini|ollama
func NewProvider(cfg Config) LLMProvider { ... }
```

Il `schema *JSONSchema` è il parametro per structured output (JSON mode). Ogni provider lo implementa secondo le proprie API:
- OpenAI: `response_format: {type: "json_schema", json_schema: schema}`
- Anthropic: prefilling `{"` + system prompt con schema esplicito
- Gemini: `generation_config.response_mime_type: "application/json"` + schema
- Ollama: `format: schema` (structured output nativo)

### 5.3 FIFO Queue Per-Player

```go
// internal/queue/playerqueue.go

type PlayerQueue struct {
    mu     sync.Mutex
    queues map[string]chan TurnRequest  // username → channel
}

type TurnRequest struct {
    Ctx     context.Context
    Input   string
    SSEChan chan<- SSEEvent  // risposta in real-time al client
}

func (q *PlayerQueue) Enqueue(username string, req TurnRequest) {
    q.mu.Lock()
    ch, ok := q.queues[username]
    if !ok {
        ch = make(chan TurnRequest, 10) // max 10 messaggi in coda
        q.queues[username] = ch
        go q.worker(username, ch) // goroutine dedicata per player
    }
    q.mu.Unlock()
    
    select {
    case ch <- req:
    default:
        // coda piena: rifiuta con errore 429
        req.SSEChan <- SSEEvent{Type: "error", Text: "La coda è piena. Attendi la risposta precedente."}
    }
}

func (q *PlayerQueue) worker(username string, ch <-chan TurnRequest) {
    for req := range ch {
        engine.ProcessTurn(req) // blocca fino al completamento
    }
}
```

Elegante e sicuro. Zero promise chaining fragile come nel Node.js prototype.

### 5.4 State Machine

```go
// internal/game/statemachine.go

type GameState string

const (
    StateWorldNavigation GameState = "WORLD_NAVIGATION"
    StateCombat          GameState = "COMBAT"
    StateDungeonExplore  GameState = "DUNGEON_EXPLORATION"
    StateDungeonCombat   GameState = "DUNGEON_COMBAT"
)

// Transizioni valide
var validTransitions = map[GameState][]GameState{
    StateWorldNavigation: {StateCombat, StateDungeonExplore},
    StateCombat:          {StateWorldNavigation},
    StateDungeonExplore:  {StateDungeonCombat, StateWorldNavigation},
    StateDungeonCombat:   {StateDungeonExplore},
}

func Transition(current, next GameState) error {
    for _, allowed := range validTransitions[current] {
        if allowed == next { return nil }
    }
    return fmt.Errorf("transizione invalida: %s → %s", current, next)
}
```

---

## 6. Agent System e Pipelines

### 6.1 GM Agent

Il GM Agent è il cuore. Costruisce il prompt, chiama l'LLM, parsa la risposta.

**Prompt Architecture (3 livelli per prefix cache)**:

```
[Livello 1 — STATICO, invariante]
  - Regole del GM (filosofia, vincoli, regole critiche)
  - Schema JSON risposta completo con descrizione campi
  - Assiomi del mondo (lore permanente, Crysta, fazioni)
  - Formule di gioco (reference)
  Dimensione target: ~1500 token | Cache hit: 100%

[Livello 2 — SEMI-STATICO, cambia max 1-2 volte a sessione]
  - Scheda personaggio (classe, skill, equipaggiamento)
  - Lore zona corrente (da world_zone catalog)
  - World flags attivi (boss sconfitti, eventi globali)
  - NPC presenti nella zona (da npc_instance)
  - RAG chunks rilevanti (top-8 da knowledge_base)
  Dimensione target: ~2000 token | Cache hit: 70-90%

[Livello 3 — DINAMICO, cambia ogni turno]
  - HP/MP/STM correnti
  - Combat state (nemico attivo, tensione tattica, party)
  - context_memo (max 800 token → compaction se supera)
  - session_log ultimi 12 messaggi (verbatim)
  - serverDirectives (pending events, restrizioni equip, weapon broken, ecc.)
  - Messaggio player corrente
  Dimensione target: ~1500 token
```

**Total**: ~5000 token, aggressivamente cached. Con provider che supportano prefix cache (DeepSeek, Anthropic, Google) il costo effettivo è ~1500 token per turno.

**Retry logic**:
```
1. Prima chiamata: prompt normale
2. Se JSON.parse() fallisce:
   → retry con system message: "Il tuo output precedente non era JSON valido.
     Ecco il buffer ricevuto: [buffer]. Correggilo e restituisci solo JSON valido."
3. Se fallisce ancora: log errore, restituisci al client errore narrativo generico
4. Se 3 turni consecutivi falliscono per lo stesso player: alert admin log
```

### 6.2 Validator Agent

Async, non blocca il turno. Viene chiamato dopo il commit dello stato, analizza la risposta GM.

**Input**: risposta GM completa, contesto del turno (input player, stato pre-turno)
**Output**: score JSON `{total: 0-100, issues: [...]}`

**Metriche verificate**:
- `narrative_length`: >= 100 chars, <= 2000 chars
- `battle_tags_plausibility`: i COMBAT_HIT_ENEMY_N sono nel range ragionevole per le stat del nemico
- `json_completeness`: tutti i campi richiesti per il contesto (es: in combat → battle_tags deve esistere)
- `money_in_state_updates`: se presente → flag violazione (questa regola viene spesso ignorata dai modelli meno capaci)
- `context_memo_growth`: se il memo è cresciuto > 20% → suggerisce compaction

I risultati vengono loggati in SurrealDB su una tabella `validation_log` per osservabilità. Se il score scende sotto 40 per 3 turni consecutivi → alert che suggerisce cambio provider LLM.

### 6.3 Context Compactor Agent

Trigger: `len(context_memo) > 800 token` oppure ogni 20 turni.

**Task**: 
1. Legge il context_memo corrente + ultime 10 diary entries
2. Chiama LLM con task: "Comprimi questo memo mantenendo tutti i fatti meccanici rilevanti (accordi, transazioni, stato trattative aperte, flags scena). Elimina ridondanze. Max 400 token."
3. Promuove a diary entry i fatti significativi (prima di compattare)
4. Sostituisce context_memo con la versione compressa
5. Genera embedding del riassunto e lo aggiunge alla knowledge_base

### 6.4 Content Generator Agent

**Distinzione critica — contenuto narrativo vs contenuto meccanico**

Il sistema distingue due categorie di contenuto che seguono pipeline diverse:

| Tipo | Esempi | Chi lo crea | Pipeline |
|------|--------|-------------|----------|
| **Narrativo** | Descrizioni NPC, lore zone, flavour text dungeon, dialoghi | Content Generator Agent | genera → valida → embed → persist |
| **Meccanico** | Skill stats, class tree, monster HP/drop, formule, titoli | Game Designer (tu) | definisci → valida coerenza → persist |

Il contenuto meccanico **non viene generato dall'AI**: le stat di un'abilità, i requisiti di una classe o i drop di un mostro sono decisioni di design deliberate che richiedono bilanciamento. L'AI che inventa che uno skill fa 500 danni a costo 0 rompe il gioco.

Il contenuto meccanico viene inserito tramite una **Design Pipeline**:
1. Game designer definisce i parametri (in file YAML/JSON di design oppure via admin UI)
2. Il `DesignValidator` verifica coerenza: stat nei range attesi, nessun ID duplicato, riferimenti validi tra tabelle
3. Persist in SurrealDB nei cataloghi statici

Il contenuto narrativo segue invece la pipeline AI:

Trigger:
- Player entra in una zona nuova → genera 2-3 NPC con descrizione + personalità
- Player inizia una quest → genera flavour text dettagliato per ogni obiettivo
- Admin attiva la generazione di un nuovo dungeon

**Output validato da Validator Agent** prima di essere persistito. Il Content Generator opera con un budget token separato (non conta verso il turno di gioco).

### 6.5 Dungeon Generator Agent

**Input**: template dungeon (numero stanze, bioma, difficoltà, tema)
**Output**: dungeon completo in formato SurrealDB-ready:
```json
{
  "id": "dungeon_uuid",
  "name": "Cripta delle Ossa Dimenticate",
  "location": "Aokara",
  "rooms": [
    { "id": "room_01", "type": "entrance", "name": "Ingresso",
      "description": "...", "connections": ["room_02", "room_03"],
      "monsters": null, "trap": null, "lore": "..." },
    { "id": "room_02", "type": "combat", "name": "Sala delle Guardie",
      "description": "...", "connections": ["room_01", "room_04"],
      "monsters": [{"id": "goblin_guerriero", "count": 3}], "trap": null }
  ]
}
```

Il validatore verifica:
- Connettività del grafo (nessuna stanza isolata)
- Presenza di almeno 1 stanza boss e 1 stanza reward
- Bilanciamento dei tipi (non più del 60% di stanze combat)

---

## 7. MCP Tools

L'MCP server è implementato in Go (o Node.js se si preferisce eco-sistema più ricco) e espone tool al GM Agent e agli altri agenti.

### 7.1 Tool Definiti

**Tool 1: `query_knowledge_base`**
```json
{
  "name": "query_knowledge_base",
  "description": "Cerca nella knowledge base del mondo e del personaggio usando semantic search. Usa per recuperare lore di zone, dettagli NPC, eventi passati, assiomi del mondo.",
  "inputSchema": {
    "query": "string — cosa stai cercando",
    "scope": "global | character | all",
    "limit": "int (default 5, max 10)"
  }
}
```

**Tool 2: `get_game_state_summary`**
```json
{
  "name": "get_game_state_summary",
  "description": "Recupera un sommario leggibile dello stato corrente del gioco per un personaggio. Read-only.",
  "inputSchema": {
    "character_id": "string",
    "fields": ["character", "inventory", "session", "npcs", "quests"]
  }
}
```

**Tool 3: `search_web`**
```json
{
  "name": "search_web",
  "description": "Cerca sul web. Usa per trovare ispirazione narrativa, dettagli di ambientazione fantasy, riferimenti culturali. NON usare per ricerche personali sul player.",
  "inputSchema": {
    "query": "string"
  }
}
```

**Tool 4: `roll_dice`**
```json
{
  "name": "roll_dice",
  "description": "Tira dadi con seed deterministico basato su turn_id. Per decisioni narrative che non sono meccanicamente risolte dai battle tags.",
  "inputSchema": {
    "notation": "string — es: '2d6+3', '1d20', 'd100'",
    "purpose": "string — descrizione del tiro"
  }
}
```

**Tool 5: `generate_npc_description`**
```json
{
  "name": "generate_npc_description",
  "description": "Genera una descrizione dettagliata per un NPC nuovo. Chiama il Content Generator Agent internamente.",
  "inputSchema": {
    "npc_type": "string — tipo (fabbro, mercante, antagonista...)",
    "faction": "string",
    "zone": "string",
    "personality_hints": "array<string>"
  }
}
```

**Tool 6: `validate_json_response`**
```json
{
  "name": "validate_json_response",
  "description": "Valida un JSON response draft contro lo schema GM prima di restituirlo. Usa prima di finalizzare la risposta.",
  "inputSchema": {
    "response_draft": "object"
  }
}
```

### 7.2 Implementazione MCP Server

Il MCP server in Go espone i tool via stdio (per uso locale con agenti Claude) o via HTTP + SSE (per agenti remoti). Usa l'[MCP Go SDK](https://github.com/mark3labs/mcp-go).

```go
// cmd/mcp-server/main.go

func main() {
    s := server.NewMCPServer("aether-horizon-tools", "1.0.0")
    
    s.AddTool(tools.QueryKnowledgeBase(dbClient, embedder))
    s.AddTool(tools.GetGameStateSummary(dbClient))
    s.AddTool(tools.SearchWeb(httpClient))
    s.AddTool(tools.RollDice())
    s.AddTool(tools.GenerateNPCDescription(contentGenAgent))
    s.AddTool(tools.ValidateJSONResponse(schema))
    
    server.ServeStdio(s)
}
```

---

## 8. Frontend React

### 8.1 Struttura Directory

```
frontend/
├── src/
│   ├── main.jsx              # entry point React
│   ├── App.jsx               # router + auth guard + WS connection
│   ├── store/
│   │   ├── gameStore.js      # Zustand store per stato gioco
│   │   ├── uiStore.js        # Zustand store per UI state (modali, layout)
│   │   └── authStore.js      # Zustand store per auth (JWT token)
│   ├── api/
│   │   ├── client.js         # fetch wrapper con auth header + error handling
│   │   ├── chat.js           # SSE streaming hook
│   │   └── endpoints.js      # costanti URL endpoint
│   ├── panels/               # Panel registry — ogni file è un panel autonomo
│   │   ├── PanelRegistry.jsx # registry: {id, name, component, defaultSize}
│   │   ├── ChatPanel.jsx     # chat + narrativa streaming
│   │   ├── StatsPanel.jsx    # HP/MP/STM + reputazione + status effects
│   │   ├── InventoryPanel.jsx # borsa + equipaggiamento
│   │   ├── SkillPanel.jsx    # albero skill + loadout
│   │   ├── QuestPanel.jsx    # tracker quest + bounties
│   │   ├── MapPanel.jsx      # mappa mondo SVG / mappa dungeon BFS
│   │   ├── NPCPanel.jsx      # NPC persistenti + modal dettaglio
│   │   ├── BestiaryPanel.jsx # bestiario
│   │   ├── DiaryPanel.jsx    # diario di viaggio
│   │   └── CombatPanel.jsx   # stato combattimento + tension bar + party
│   ├── dashboard/
│   │   ├── DashboardBuilder.jsx # react-grid-layout wrapper
│   │   ├── PanelWrapper.jsx     # header panel con drag handle + collapse
│   │   ├── LayoutPresets.jsx    # preset layout (default, combat focus, exploration)
│   │   └── useDashboard.js      # hook per persist/load layout da SurrealDB
│   ├── components/           # componenti UI shared (shadcn/ui based)
│   │   ├── ui/               # re-export shadcn components
│   │   ├── StatBar.jsx       # barra HP/MP/STM con animazione
│   │   ├── StatusPill.jsx    # pill per status effect
│   │   ├── ItemCard.jsx      # card oggetto in borsa
│   │   ├── SkillSlot.jsx     # slot skill con cooldown overlay
│   │   └── ToastEngine.jsx   # sistema toast per UI events (level_up, quest, ecc.)
│   ├── hooks/
│   │   ├── useSSE.js         # EventSource hook con reconnect automatico
│   │   ├── useWebSocket.js   # WebSocket hook per global announcements
│   │   └── useGameState.js   # sync stato gioco da /api/state
│   └── styles/
│       ├── globals.css       # reset + variabili CSS tema
│       ├── theme.css         # dark theme Overflow (ispirato al logo)
│       └── animations.css    # CSS animations (shake, flash, pulse, overdrive glow)
├── index.html
├── vite.config.js
└── package.json
```

### 8.2 Dashboard Builder

Il sistema di panel è costruito su `react-grid-layout`:

```jsx
// dashboard/DashboardBuilder.jsx

import GridLayout from 'react-grid-layout';
import { PanelRegistry } from '../panels/PanelRegistry';
import { useDashboard } from './useDashboard';

export function DashboardBuilder() {
  const { layout, updateLayout, visiblePanels } = useDashboard();
  
  return (
    <GridLayout
      layout={layout}
      cols={12}            // griglia 12 colonne
      rowHeight={60}
      onLayoutChange={updateLayout}  // persiste su SurrealDB
      draggableHandle=".panel-drag-handle"
    >
      {visiblePanels.map(panelId => {
        const Panel = PanelRegistry[panelId].component;
        return (
          <div key={panelId}>
            <PanelWrapper id={panelId}>
              <Panel />
            </PanelWrapper>
          </div>
        );
      })}
    </GridLayout>
  );
}
```

**Layout presets** selezionabili:
- **Default**: 3 colonne (stats | chat | inventory)
- **Combat Focus**: chat più grande, combat panel in evidenza, stats sempre visibili
- **Exploration**: mappa grande, diary + quest panel visibili, inventario collassato
- **Mobile**: 1 colonna, solo chat + stats essenziali

### 8.3 Chat Panel con Streaming

```jsx
// panels/ChatPanel.jsx

function ChatPanel() {
  const { messages, sendMessage, isStreaming } = useChatSSE();
  
  return (
    <div className="chat-panel">
      <ScrollArea>
        {messages.map(msg => (
          msg.role === 'gm'
            ? <GMBubble key={msg.id} text={msg.text} isStreaming={msg.isStreaming} />
            : <PlayerBubble key={msg.id} text={msg.text} />
        ))}
      </ScrollArea>
      <ChatInput onSubmit={sendMessage} disabled={isStreaming} />
    </div>
  );
}

// GMBubble usa react-markdown con syntax highlight per il testo narrativo
// Il testo appare progressivamente con cursor blinking durante lo streaming
```

### 8.4 UI Events System

```js
// components/ToastEngine.jsx
// Gestisce tutti gli ui_events emessi dal server nel payload "done"

const UI_EVENT_HANDLERS = {
  level_up:         () => showToast("LEVEL UP!", "gold"),
  skill_unlocked:   (e) => showToast(`Skill sbloccata: ${e.skill_name}`, "blue"),
  quest_completed:  (e) => showToast(`Quest completata: ${e.quest_name}`, "green"),
  SCREEN_SHAKE:     () => triggerAnimation("screen-shake", 400),
  RED_FLASH:        () => triggerAnimation("red-flash", 300),
  GOLDEN_GLOW:      () => triggerAnimation("golden-glow", 600),
  PLAYER_DEATH:     () => triggerDeathScreen(),  // dissolvenza nera + GAME OVER
  BOSS_PHASE_2:     () => { showToast("FASE 2!", "red"); triggerAnimation("boss-shake", 600); },
  WEAPON_BROKEN:    () => showToast("Arma spezzata!", "orange"),
  PART_BROKEN:      (e) => showPartBreakToast(e.part_name),
  TRANSACTION_FAILED: () => showToast("Fondi insufficienti", "red"),
};
```

---

## 9. Roadmap Implementativa

### Fase 0 — Setup ✅ COMPLETATA
- [x] Struttura directory (backend Go + frontend React separati) in `Project-Beyond/`
- [x] Docker Compose: SurrealDB (`memory` in dev, `surrealkv` in prod)
- [x] Schema SurrealDB v1 (tabella `user`, IF NOT EXISTS idempotente)
- [x] Auth JWT completa (register, login, refresh, /me) — testata e funzionante
- [x] Vite + React + Tailwind, route `/login` e `/game` con auth guard
- [x] Repo Git: da creare quando il nome del progetto è definito
- **Nota**: i cataloghi statici (skill, classi, mostri) NON vengono importati da Shanfro.
  Il contenuto meccanico viene creato tramite Design Pipeline (sezione 6.4).
  Il contenuto narrativo tramite Content Generator Agent (Phase 3-4).
  Per Phase 1 si usa un dataset minimo direttamente in SurrealDB.

**Bug corretti durante il setup:**
- `file://` → `surrealkv://` (schema storage SurrealDB v3)
- `SELECT 1` → `RETURN 1;` (SurrealQL non supporta bare SELECT)
- Path schema: `../../..` → `../..` (livelli directory errati)
- `DEFINE NAMESPACE/DATABASE ON NAMESPACE` → sintassi v3 corretta
- `type::thing()` → `type::record()` (rinominata in SurrealDB v3)
- `IF NOT EXISTS` su tutti i DEFINE per idempotenza reale

### Fase 1 — Core Turn Engine ✅ COMPLETATA
- [x] Interfaccia LLMProvider + provider OpenAI-compatible (DeepSeek default)
- [x] Player FIFO queue (goroutine + channel)
- [x] `buildSystemPrompt()` in Go (3 livelli, prefix cache)
- [x] GM Agent: call + streaming SSE + JSON parse + retry
- [x] Battle Tags Engine
- [x] State machine (4 stati, transizioni validate)
- [x] Endpoint `POST /api/chat` con SSE, `GET /api/state`
- [x] Frontend: ChatPanel con streaming, StatsPanel base, connessione SSE

### Fase 2 — Sistemi di Gioco ✅ COMPLETATA (parziale)
- [x] Level up, skill unlocks, titoli
- [x] Quest tracker (start/complete/fail/progress)
- [x] Tactical tension bar + overdrive
- [x] Loot system (tier normal/elite/boss)
- [x] Inventario + equipaggiamento (equip/unequip)
- [x] Stat point allocation (PUT /api/character/stats)
- [x] Frontend: InventoryPanel, SkillPanel, QuestPanel, LootPopup, TacticalTensionBar, StatAllocModal
- [x] Dungeon engine: GenerateDungeon, MoveInDungeon, DiscoveredRooms + API `/dungeon/*`
- [x] Transaction ledger: `ledger.go` + tabella `transaction_log`, scrittura asincrona ad ogni cambio oro
- [x] Death/Respawn: UIEvent `DEATH` inoltrato al frontend, `near_death_survives` incrementato, `PlayerDied` in DonePayload
- [ ] Transaction ledger, multi-phase boss, death & respawn — rimandati

### Fase 3 — RAG e Knowledge Base ✅ COMPLETATA
- [x] SurrealDB schema knowledge_base con HNSW 1536 dim + BM25
- [x] Embedding provider OpenAI-compatible (disabilitato default, DeepSeek non supporta embeddings)
- [x] 14 documenti lore seedati (world, NPC, meccaniche)
- [x] RAG retriever: hybrid HNSW + BM25 con RRF merge
- [x] Integrazione nel turn engine (step 3: Retrieve → inject in Level 3 prompt)

### Fase 4 — Agent System e Pipelines ✅ COMPLETATA (parziale)
- [x] Validator Agent: scoring asincrono post-turno + validation_log
- [x] Context Compactor Agent: trigger su memo > 800 token, LLM async
- [x] WebSocket broadcast hub (gorilla/websocket) + GET /ws + AnnouncementToasts
- [ ] Content Generator Agent — rimandato
- [ ] Dungeon Generator Agent — rimandato
- [ ] MCP Tools Server — rimandato

### Fase 5 — Dashboard Builder e UX ✅ COMPLETATA
- [x] react-grid-layout v2 integration (`ResponsiveGridLayout`)
- [x] Panel Registry con tutti i panel autonomi (`PanelRegistry.js`)
- [x] Layout presets (default, combat, exploration)
- [x] Persist layout per personaggio (localStorage, chiave `daydream-layout-<charId>`)
- [x] Panel show/hide/reset UI nella toolbar
- [x] Tutti i toast e UI animations (SCREEN_SHAKE, RED_FLASH, OVERDRIVE, DEATH) — `AnimationLayer.jsx`
- [x] Tema dark Daydream (`globals.css`)
- [x] WebSocket broadcast hub + toast annunci (`useWebSocket.js`, `AnnouncementToasts`)
- [x] Allocazione stat points (`StatAllocModal`, `PUT /api/character/stats`)

**Milestone**: interfaccia responsive, personalizzabile, con tutti i panel operativi. ✅

### Fase 6 — Polish e Deploy ✅ COMPLETATA
- [x] Rate limiting: token bucket in-memory per-utente su `POST /api/chat` (5s min interval, `ChatRateLimit` middleware)
- [x] Structured logging: `log/slog` con handler JSON/text configurabile via `LOG_FORMAT`
- [x] Docker multi-stage build: `backend/Dockerfile`, `frontend/Dockerfile` (nginx + SPA fallback)
- [x] `docker-compose.yml` completo con SurrealDB + backend + frontend su rete `daydream`
- [x] `.env.example` completamente documentato con commenti e istruzioni
- [x] Provider Anthropic in LLM adapter (`internal/llm/anthropic.go`), selezionabile via `LLM_PROVIDER=anthropic`
- [x] `.dockerignore` per backend e frontend
- [x] Input sanitization: strip HTML/script injection e caratteri di controllo sui messaggi in ingresso (`sanitize.go`)

**Milestone**: sistema pronto per deploy su VPS/cloud. ✅

---

### Fase 7 — Sistemi Avanzati di Gioco ✅ COMPLETATA

#### 7.1 World Flags
Sistema di stato narrativo persistente scritto dal GM in tempo reale.
- **Modello**: `WorldFlag {character_id, scope, key, value, updated_at}` su tabella SCHEMALESS
- **Scope**: `world` | `kingdom` | `city` | `npc` | `faction` | `dungeon` | `player`
- **Flusso**: il GM emette `world_flags: [{scope, key, value}]` → `UpsertWorldFlags` → SurrealDB
- **Prompt injection**: `LoadRelevantFlags` carica i flag rilevanti per location corrente + player scope ad ogni turno
- **Frontend**: `WorldPanel` raggruppa i flag per scope con colori distinti (player scope in ciano)
- **File**: `game/flagengine.go`, `models/world.go`, `api/world.go`, `panels/WorldPanel.jsx`

#### 7.2 Specializzazioni di Classe
Tre tier di scelta identitaria per ogni classe (5 classi × 3 tier × 2 opzioni = 30 specializzazioni).
- **Livelli di sblocco**: Tier 1 → Lv **25**, Tier 2 → Lv **50**, Tier 3 → Lv **80**
- **Struttura**: ogni opzione ha flavor narrativo, stat bonus passivo, effetto descritto
- **Storage**: `chosen_specs []string` sul personaggio
- **UI**: `SpecChoiceModal` — modale bloccante con 2 opzioni al raggiungimento del livello soglia
- **File**: `game/specializations.go`, `api/specialization.go`, `components/SpecChoiceModal.jsx`

#### 7.3 Albero Skill
80 skill totali (5 classi × 4 rami × 4 nodi) con costi incrementali e prerequisiti.
- **Costi per tier**: T1=3pt, T2=7pt, T3=15pt, T4=28pt (capstone)
- **Punti skill**: **3 punti/livello** (accumulati da level-up, backfill automatico per personaggi esistenti)
- **Skill custom livellabili**: il GM può concedere abilità uniche (Level 1-5), il giocatore le potenzia spendendo punti (15→30→50→80 per livello)
- **Storage**: `skill_tree_unlocks []string`, `tree_points_available int`, `custom_skills []GMCustomSkill`
- **UI**: `SkillTreePanel` con branch expandibili, costi visibili, sezione "Abilità Uniche" separata con pulsante Potenzia
- **File**: `game/skilltree.go`, `api/skilltree.go`, `panels/SkillTreePanel.jsx`

#### 7.4 Loot Procedurale
Sistema di generazione item influenzato da LUC (rarità/drop rate) e TEC (affix magici).
- **Rarità**: common / uncommon / rare / epic / legendary con pesi LUC-dipendenti
- **Affix**: count influenzato da TEC; nomi e stat generati proceduralmente
- **Item non identificati**: `appraised: false` su tutti i drop non-common; stat oscurate nell'UI fino all'identificazione
- **File**: `game/loot.go`

#### 7.5 Mercato e Contrattazione
Mercato generato proceduralmente per ogni location con NPC bargaining LUC/TEC-dipendente.
- **Generazione**: 4-6 item per mercato, rarità minima uncommon, influenzata da LUC del personaggio
- **Item opachi**: tutti gli item del mercato nascono `analyzed: false` — il giocatore vede solo tipo e prezzo, non stats né rarità
- **Analisi**: `POST /api/market/analyze` — accuracy TEC+LUC dipendente (formula dinamica), può dare stat percepite errate
- **Contrattazione**: `NegotiatePrice` — successo basato su LUC (20%+3%/pt sopra 5), TEC bonus per item magici
- **UI**: `MarketPanel` con item misteriosi, pulsante Analizza, stat percepite con "~" giallo
- **File**: `game/market.go`, `api/market.go`, `models/market.go`, `panels/MarketPanel.jsx`

#### 7.6 Identificazione Item — Formula Dinamica
L'identificazione usa TEC+LUC con formula scalata per livelli 1-100.
```
effective_tec = TEC + LUC × 0.3
required_tec  = item_level × rarity_multiplier
               (common:×0.5, uncommon:×0.8, rare:×1.3, epic:×2.0, legendary:×3.0)

costo = item_level × rarity_mult_gold × 15
      → sconto 50% se effective_tec ≥ required_tec × 0.5
      → sconto 75% se effective_tec ≥ required_tec

accuracy (stat percepite):
  effective_tec 0-10  → 20-50% (errori grandi, rarità può shiftare ±1 tier)
  effective_tec 11-30 → 50-75%
  effective_tec 31-60 → 75-90%
  effective_tec 61+   → 90-100%
```
- **File**: `api/inventory.go` (AppraiseItem), `game/market.go` (ApplyTecAnalysis)

---

### Fase 8 — Bilanciamento e Creazione Personaggio ✅ COMPLETATA

#### 8.1 Creazione Personaggio con Pool Stat
- **Base**: 5 per ogni stat (non 3)
- **Pool distribuibile**: 15 punti (non 20)
- **Bonus classe**: +3 a 2 stat (non +2)
- **UI a 2 step**: Step 1 = nome + classe con badge bonus visibili; Step 2 = allocazione con counter e pool rimanente in tempo reale
- **Stat descriptions**: mostrate durante l'allocazione (STR=mischia, DEX=ranged, AGI=velocità, TEC=magia/ID, VIT=HP, LUC=fortuna)

#### 8.2 Bilanciamento Scale Livelli 1-100
- **Stat point/livello**: 5 (non 1 o 3)
- **Skill point/livello**: 3 (non 2)
- **Specializzazioni**: Lv 25/50/80 (non 3/7/12)
- **Identificazione**: soglie dinamiche basate su `item_level × rarity_mult` (non TEC fisse)

---

### Fase 9 — Sistema Tempo e Quest Viventi ✅ COMPLETATA

#### 9.1 Orologio In-Game
- **Struttura**: `GameTime {Day, Hour, Minute}` sulla `GameSession`; inizia a Giorno 1, 08:00
- **Avanzamento**: il GM emette `action_category` (conversation/combat/exploration/travel_local/travel_regional/travel_long/rest/crafting); il backend calcola i minuti automaticamente senza lasciare la scelta al modello
- **Calcolo minuti** (`game/timecalculator.go`):

| Category | Base | Modificatori |
|---|---|---|
| conversation | 25 min | — |
| combat | 20 min | +15 per nemico sconfitto (da battle tags) |
| exploration | 30 min | +20 per stanza dungeon attraversata |
| travel_local | 20 min | — |
| travel_regional | 240 min | — |
| travel_long | 1440 min | — |
| rest | 420 min | azzera HoursAwake |
| crafting | 60 min | — |

- **Minimo**: 10 min per turno anche senza categoria esplicita
- **Frontend**: clock widget in topbar — `☀ Giorno 3  14:30` / `🌙 Giorno 3  23:15`
- **File**: `models/gametime.go`, `game/timecalculator.go`

#### 9.2 Ciclo Giorno/Notte
- `TimeOfDay()` → morning (06-12) / afternoon (12-17) / evening (17-21) / night (21-06)
- Il GM riceve l'ora corrente nel prompt e può narrare l'inaccessibilità di mercanti/NPC di notte
- World flag `time_of_day` aggiornato automaticamente

#### 9.3 Sleep Deprivation
- `HoursAwake float64` tracciato sulla sessione, incrementato ad ogni avanzamento del clock
- Debuff automatici via `StatusEffect` (rimossi solo con `rest`):

| Ore sveglio | Effetto |
|---|---|
| < 20h | Nessuno |
| 20–28h | **Stanco** — tutte le stat −2 |
| 28–36h | **Esausto** — stat −5, HP max −15%, MP max −30% |
| 36–48h | **Al limite** — stat −10, HP max −30% |
| 48h+ | **Collasso imminente** — stat −10, HP max −30%, flag critico al GM |

- **File**: `game/timecalculator.go` (ApplySleepDeprivation, SleepDeprivationDebuffs)

#### 9.4 Quest Viventi
Nuova struttura Quest con ciclo di vita completo:

```go
type Quest struct {
    // Campi base
    ID, Title, Description, GiverNPC, Category string
    Difficulty  int    // 1-5
    Urgency     string // low/medium/high/critical
    Objectives  []QuestObjective
    Rewards     QuestReward
    Status      string // active/completed/failed/expired

    // Tempo in-game
    DeadlineDay, DeadlineHour int  // scadenza assoluta in ore di gioco

    // Escalation
    EscalationStage int
    Escalations     []QuestEscalation  // {Stage, Description, WorldFlagKey, TriggerAtPercent}
    ConsequenceOnFail string
}
```

#### 9.5 Pipeline di Bilanciamento Quest (`game/questbalancer.go`)
Il GM dichiara difficoltà e urgency; il pipeline calcola meccanicamente:

| Difficoltà | Gold mult | Exp mult | Item max | Tempo base/stage |
|---|---|---|---|---|
| 1 Triviale | ×0.5 | ×0.5 | Common | — |
| 2 Facile | ×1.0 | ×1.0 | Uncommon | — |
| 3 Normale | ×2.0 | ×2.0 | Rare | — |
| 4 Difficile | ×4.0 | ×4.0 | Epic | — |
| 5 Epico | ×8.0 | ×8.0 | Legendary | — |

Urgency base ore per stage: low=25h, medium=14h, high=7h, critical=3h (×1.4 per diff 5, ×0.8 per diff 1).

**Hard limits**: gold cap = `charLevel × 25 × mult`, exp cap = `charLevel × 40 × mult`, rarità item cappata per difficoltà.

#### 9.6 Motore Escalation (`game/questescalation.go`)
Ad ogni turno, per ogni quest attiva:
1. Calcola % tempo consumato (`now / deadline`)
2. Se % supera `TriggerAtPercent` dello stage → avanza `EscalationStage`
3. Emette world flag automatico (`{key: "quest_<id>_stage", value: "N"}`)
4. Se `now >= deadline` → status `expired`, spostata in `QuestsCompleted`, emesso flag conseguenza

#### 9.7 Rewards Automatiche
Al completamento quest (GM emette `complete: "quest_id"`):
- Gold applicato immediatamente a `character.Money`
- Exp processato via `applyExp()` (può triggerare level-up)
- Item inseriti in `inventory.Bag`
- **Il GM NON deve emettere `GOLD_GAIN`/`EXP_GAIN` separati** — il sistema applica automaticamente

#### 9.8 UI QuestPanel Aggiornata
- Badge difficoltà (★ da 1 a 5)
- Colore urgency (grigio/giallo/arancio/rosso)
- Countdown `⏱ 1g 6h` o `⏱ 4h` in tempo reale
- Warning escalation stage corrente (box arancio con descrizione)
- Item reward visibili come badge
- Cronologia include status `expired`

---

## Appendice A — Dipendenze Go principali

```go
// go.mod essenziale
require (
    github.com/gin-gonic/gin          v1.10.x  // HTTP framework
    github.com/surrealdb/surrealdb.go v2.x     // SurrealDB client
    github.com/golang-jwt/jwt/v5      v5.x     // JWT
    golang.org/x/crypto               latest   // bcrypt
    github.com/mark3labs/mcp-go       latest   // MCP server/client
    github.com/gorilla/websocket      v1.5.x   // WebSocket broadcast
    github.com/tiktoken-go/tokenizer  latest   // stima token count per compaction trigger
)
```

## Appendice B — Dipendenze Node/React principali

```json
{
  "dependencies": {
    "react": "^18",
    "react-dom": "^18",
    "react-router-dom": "^6",
    "zustand": "^4",
    "react-grid-layout": "^1.4",
    "react-markdown": "^9",
    "remark-gfm": "^4",
    "@radix-ui/react-*": "latest",  // via shadcn/ui
    "tailwindcss": "^3",
    "class-variance-authority": "latest",
    "clsx": "latest",
    "sonner": "latest"  // toast
  },
  "devDependencies": {
    "vite": "^5",
    "@vitejs/plugin-react": "^4"
  }
}
```

## Appendice C — Variabili d'Ambiente

```env
# Server
PORT=8080
ENV=development

# SurrealDB
SURREAL_URL=ws://localhost:8000/rpc
SURREAL_USER=root
SURREAL_PASS=root
SURREAL_NS=aether
SURREAL_DB=horizon

# JWT
JWT_SECRET=<random 64 byte hex>
JWT_EXPIRY=24h

# LLM
LLM_PROVIDER=openai              # openai | anthropic | gemini | ollama
LLM_API_KEY=<api_key>
LLM_BASE_URL=https://api.openai.com/v1  # per openai-compatible
LLM_MODEL=gpt-4o-mini
LLM_TEMPERATURE=0.7

# Embedding
EMBED_PROVIDER=jina              # jina | openai
EMBED_MODEL=jina-embeddings-v3
EMBED_API_KEY=<api_key>
EMBED_DIMENSIONS=768

# Feature flags
MAX_CONTEXT_TOKENS=8000          # limite context window configurabile
MAX_SESSION_HISTORY=12           # turni verbatim in session_log
CONTEXT_MEMO_MAX_TOKENS=800      # trigger compaction
RATE_LIMIT_CHAT=1                # req/2s per /api/chat
```
