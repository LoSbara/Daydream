-- Daydream — SurrealDB Schema
-- Phase 0: user | Phase 1: character, inventory, game_session | Phase 3: knowledge_base
-- Idempotente: IF NOT EXISTS su ogni DEFINE.

-- ============================================================
-- USERS (Phase 0)
-- ============================================================
DEFINE TABLE IF NOT EXISTS user SCHEMAFULL;
DEFINE FIELD IF NOT EXISTS username      ON user TYPE string
    ASSERT string::len($value) >= 3 AND string::len($value) <= 32;
DEFINE FIELD IF NOT EXISTS password_hash ON user TYPE string;
DEFINE FIELD IF NOT EXISTS created_at    ON user TYPE datetime;
DEFINE FIELD IF NOT EXISTS last_login    ON user TYPE option<datetime>;
DEFINE INDEX IF NOT EXISTS idx_username ON user FIELDS username UNIQUE;

-- ============================================================
-- CHARACTER (Phase 1)
-- SCHEMALESS: il modello di gioco ha oggetti annidati complessi.
-- Gli indici garantiscono l'unicità e le performance di lookup.
-- ============================================================
DEFINE TABLE IF NOT EXISTS character SCHEMALESS;
DEFINE FIELD IF NOT EXISTS user_id               ON character TYPE string;
DEFINE FIELD IF NOT EXISTS name                  ON character TYPE string;
DEFINE FIELD IF NOT EXISTS job                   ON character TYPE string;
DEFINE FIELD IF NOT EXISTS subclass              ON character TYPE option<string>;
DEFINE FIELD IF NOT EXISTS advanced_class        ON character TYPE option<string>;
DEFINE FIELD IF NOT EXISTS level                 ON character TYPE int    DEFAULT 1;
DEFINE FIELD IF NOT EXISTS experience            ON character TYPE int    DEFAULT 0;
DEFINE FIELD IF NOT EXISTS experience_to_next    ON character TYPE int    DEFAULT 100;
DEFINE FIELD IF NOT EXISTS stats                 ON character TYPE object;
DEFINE FIELD IF NOT EXISTS stat_points_available ON character TYPE int    DEFAULT 0;
DEFINE FIELD IF NOT EXISTS money                 ON character TYPE int    DEFAULT 500;
DEFINE FIELD IF NOT EXISTS skill_slots           ON character TYPE int    DEFAULT 4;
DEFINE FIELD IF NOT EXISTS titles                ON character TYPE array  DEFAULT [];
DEFINE FIELD IF NOT EXISTS status_effects        ON character TYPE array  DEFAULT [];
DEFINE FIELD IF NOT EXISTS reputation            ON character TYPE object;
DEFINE FIELD IF NOT EXISTS flags                 ON character TYPE object DEFAULT {};
DEFINE FIELD IF NOT EXISTS action_counters       ON character TYPE object;
DEFINE FIELD IF NOT EXISTS skill_cooldowns       ON character TYPE object DEFAULT {};
DEFINE INDEX IF NOT EXISTS idx_char_user ON character FIELDS user_id UNIQUE;

-- ============================================================
-- INVENTORY (Phase 1)
-- ============================================================
DEFINE TABLE IF NOT EXISTS inventory SCHEMALESS;
DEFINE FIELD IF NOT EXISTS character_id                ON inventory TYPE string;
DEFINE FIELD IF NOT EXISTS equipped                    ON inventory TYPE object;
DEFINE FIELD IF NOT EXISTS stat_bonuses_from_equipment ON inventory TYPE object;
DEFINE FIELD IF NOT EXISTS bag                         ON inventory TYPE array DEFAULT [];
DEFINE INDEX IF NOT EXISTS idx_inv_char ON inventory FIELDS character_id UNIQUE;

-- ============================================================
-- GAME SESSION (Phase 1)
-- ============================================================
DEFINE TABLE IF NOT EXISTS game_session SCHEMALESS;
DEFINE FIELD IF NOT EXISTS character_id             ON game_session TYPE string;
DEFINE FIELD IF NOT EXISTS location                 ON game_session TYPE string DEFAULT "Nexus";
DEFINE FIELD IF NOT EXISTS sub_location             ON game_session TYPE string DEFAULT "";
DEFINE FIELD IF NOT EXISTS zone_type                ON game_session TYPE string DEFAULT "safe_zone";
DEFINE FIELD IF NOT EXISTS game_state               ON game_session TYPE string DEFAULT "WORLD_NAVIGATION";
DEFINE FIELD IF NOT EXISTS combat_active            ON game_session TYPE bool   DEFAULT false;
DEFINE FIELD IF NOT EXISTS current_enemy            ON game_session TYPE option<object>;
DEFINE FIELD IF NOT EXISTS tactical_tension         ON game_session TYPE int    DEFAULT 0;
DEFINE FIELD IF NOT EXISTS skill_loadout            ON game_session TYPE array  DEFAULT [];
DEFINE FIELD IF NOT EXISTS session_log              ON game_session TYPE array  DEFAULT [];
DEFINE FIELD IF NOT EXISTS context_memo             ON game_session TYPE string DEFAULT "";
DEFINE FIELD IF NOT EXISTS pending_narrative_events ON game_session TYPE array  DEFAULT [];
DEFINE FIELD IF NOT EXISTS quests_active            ON game_session TYPE array  DEFAULT [];
DEFINE FIELD IF NOT EXISTS quests_completed         ON game_session TYPE array  DEFAULT [];
DEFINE FIELD IF NOT EXISTS turn_id                  ON game_session TYPE int    DEFAULT 0;
DEFINE FIELD IF NOT EXISTS gold_lose_cooldown_until ON game_session TYPE int    DEFAULT 0;
DEFINE FIELD IF NOT EXISTS last_gold_transaction    ON game_session TYPE string DEFAULT "";
DEFINE INDEX IF NOT EXISTS idx_session_char ON game_session FIELDS character_id UNIQUE;

-- ============================================================
-- KNOWLEDGE BASE (Phase 3)
-- Documenti lore + NPC + meccaniche con embedding vettoriale.
-- HNSW per ANN similarity search, BM25 per fulltext keyword.
-- Le dimensioni del vettore dipendono dal provider embedding:
--   nomic-embed-text (Ollama): 768 | text-embedding-3-small: 1536
-- ============================================================
DEFINE TABLE IF NOT EXISTS knowledge_base SCHEMALESS;
DEFINE FIELD IF NOT EXISTS title     ON knowledge_base TYPE string;
DEFINE FIELD IF NOT EXISTS content   ON knowledge_base TYPE string;
DEFINE FIELD IF NOT EXISTS category  ON knowledge_base TYPE string; -- world_lore | npc | mechanics | zone | diary
DEFINE FIELD IF NOT EXISTS embedding ON knowledge_base TYPE array;

-- Indice HNSW per ANN search (cosine similarity)
-- DIMENSIONE: deve corrispondere a EMBED_DIMENSIONS nel .env
-- OpenAI text-embedding-3-small: 1536 | nomic-embed-text: 768
-- Se si cambia modello, droppare e ricreare l'indice.
-- Dimensione deve corrispondere a EMBED_DIMENSIONS nel .env (Jina = 1024, OpenAI = 1536)
DEFINE INDEX IF NOT EXISTS idx_kb_hnsw ON knowledge_base FIELDS embedding HNSW DIMENSION 1024 DIST COSINE TYPE F32;

-- Nota: BM25 fulltext index rimosso — questa versione di SurrealDB non supporta
-- SEARCH ANALYZER nella DEFINE INDEX. Il RAG usa solo HNSW (vector similarity).

-- ============================================================
-- VALIDATION LOG (Phase 4)
-- Score qualità delle risposte GM per osservabilità.
-- ============================================================
DEFINE TABLE IF NOT EXISTS validation_log SCHEMALESS;
DEFINE FIELD IF NOT EXISTS session_id      ON validation_log TYPE string;
DEFINE FIELD IF NOT EXISTS turn_id         ON validation_log TYPE int;
DEFINE FIELD IF NOT EXISTS total           ON validation_log TYPE int;
DEFINE FIELD IF NOT EXISTS issues          ON validation_log TYPE array  DEFAULT [];
DEFINE FIELD IF NOT EXISTS suggest_compact ON validation_log TYPE bool   DEFAULT false;
DEFINE FIELD IF NOT EXISTS created_at      ON validation_log TYPE datetime;
DEFINE INDEX IF NOT EXISTS idx_vallog_session ON validation_log FIELDS session_id;

-- ============================================================
-- TRANSACTION LOG (Phase 2 revisited)
-- Ledger immutabile di tutti i movimenti di oro del personaggio.
-- ============================================================
DEFINE TABLE IF NOT EXISTS transaction_log SCHEMALESS;
DEFINE FIELD IF NOT EXISTS character_id  ON transaction_log TYPE string;
DEFINE FIELD IF NOT EXISTS turn_id       ON transaction_log TYPE int;
DEFINE FIELD IF NOT EXISTS type          ON transaction_log TYPE string; -- gain | loss | penalty | loot
DEFINE FIELD IF NOT EXISTS amount        ON transaction_log TYPE int;
DEFINE FIELD IF NOT EXISTS balance_after ON transaction_log TYPE int;
DEFINE FIELD IF NOT EXISTS source        ON transaction_log TYPE string; -- battle_tag | loot | death
DEFINE FIELD IF NOT EXISTS created_at    ON transaction_log TYPE datetime;
DEFINE INDEX IF NOT EXISTS idx_txlog_char ON transaction_log FIELDS character_id;

-- ============================================================
-- WORLD FLAGS
-- ============================================================
DEFINE TABLE IF NOT EXISTS world_flags SCHEMALESS;
DEFINE FIELD IF NOT EXISTS character_id ON world_flags TYPE string;
DEFINE FIELD IF NOT EXISTS scope        ON world_flags TYPE string;
DEFINE FIELD IF NOT EXISTS key          ON world_flags TYPE string;
DEFINE FIELD IF NOT EXISTS value        ON world_flags TYPE string;
DEFINE FIELD IF NOT EXISTS description  ON world_flags TYPE string DEFAULT "";
DEFINE FIELD IF NOT EXISTS updated_at   ON world_flags TYPE datetime;
DEFINE INDEX IF NOT EXISTS idx_wflags_char ON world_flags FIELDS character_id;

-- SPECIALIZATIONS (campo su character)
DEFINE FIELD IF NOT EXISTS chosen_specs ON character TYPE array DEFAULT [];

-- SKILL TREE
DEFINE FIELD IF NOT EXISTS tree_points_available ON character TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS skill_tree_unlocks    ON character TYPE array DEFAULT [];

-- CUSTOM SKILLS (abilità narrative concesse dal GM)
DEFINE FIELD IF NOT EXISTS custom_skills ON character TYPE array DEFAULT [];

-- ============================================================
-- CATALOGHI MECCANICI (seeded all'avvio)
-- ============================================================
DEFINE TABLE IF NOT EXISTS class_catalog SCHEMALESS;
DEFINE TABLE IF NOT EXISTS monster_catalog SCHEMALESS;

-- ============================================================
-- GIORNALE DI VIAGGIO — luoghi visitati dal personaggio
-- ============================================================
DEFINE TABLE IF NOT EXISTS travel_diary SCHEMALESS;
DEFINE FIELD IF NOT EXISTS character_id ON travel_diary TYPE string;
DEFINE FIELD IF NOT EXISTS location     ON travel_diary TYPE string;
DEFINE FIELD IF NOT EXISTS note         ON travel_diary TYPE string DEFAULT "";
DEFINE FIELD IF NOT EXISTS created_at   ON travel_diary TYPE datetime;
DEFINE INDEX IF NOT EXISTS idx_diary_char ON travel_diary FIELDS character_id;

-- ============================================================
-- BESTIARIO — creature incontrate/sconfitte dal personaggio
-- ============================================================
DEFINE TABLE IF NOT EXISTS bestiary_entry SCHEMALESS;
DEFINE FIELD IF NOT EXISTS character_id        ON bestiary_entry TYPE string;
DEFINE FIELD IF NOT EXISTS monster_id          ON bestiary_entry TYPE string;
DEFINE FIELD IF NOT EXISTS name                ON bestiary_entry TYPE string;
DEFINE FIELD IF NOT EXISTS tier                ON bestiary_entry TYPE string;
DEFINE FIELD IF NOT EXISTS times_defeated      ON bestiary_entry TYPE int DEFAULT 1;
DEFINE FIELD IF NOT EXISTS first_encountered   ON bestiary_entry TYPE datetime;
DEFINE FIELD IF NOT EXISTS description         ON bestiary_entry TYPE string DEFAULT "";
DEFINE INDEX IF NOT EXISTS idx_bestiary_char ON bestiary_entry FIELDS character_id;

-- ============================================================
-- NPC — personaggi non giocanti conosciuti dal personaggio
-- ============================================================
DEFINE TABLE IF NOT EXISTS npc_instance SCHEMALESS;
DEFINE FIELD IF NOT EXISTS character_id ON npc_instance TYPE string;
DEFINE FIELD IF NOT EXISTS name         ON npc_instance TYPE string;
DEFINE FIELD IF NOT EXISTS role         ON npc_instance TYPE string DEFAULT "";
DEFINE FIELD IF NOT EXISTS description  ON npc_instance TYPE string DEFAULT "";
DEFINE FIELD IF NOT EXISTS location     ON npc_instance TYPE string DEFAULT "";
DEFINE FIELD IF NOT EXISTS disposition  ON npc_instance TYPE string DEFAULT "neutrale"; -- amichevole | neutrale | ostile
DEFINE FIELD IF NOT EXISTS last_seen    ON npc_instance TYPE datetime;
DEFINE INDEX IF NOT EXISTS idx_npc_char ON npc_instance FIELDS character_id;
