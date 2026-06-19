package game

import (
	"fmt"
	"log/slog"
	"strings"

	"daydream/internal/db"
	"daydream/internal/models"
)

// UpsertWorldFlags salva o aggiorna i flag del GM nel DB.
// Usa INSERT ... ON DUPLICATE KEY per comportamento upsert via query SurrealQL inline.
func UpsertWorldFlags(database db.DBClient, charID string, flags []models.GMWorldFlag) {
	for _, f := range flags {
		scope := normalizeScope(f.Scope)
		key := normalizeScope(f.Key)
		value := strings.ReplaceAll(f.Value, "'", "\\'")
		description := strings.ReplaceAll(f.Description, "'", "\\'")

		// SurrealDB: UPSERT con WHERE clause — cerca un record esistente e lo aggiorna,
		// altrimenti ne crea uno nuovo.
		sql := fmt.Sprintf(
			`UPSERT world_flags SET character_id='%s', scope='%s', key='%s', value='%s', description='%s', updated_at=time::now() WHERE character_id='%s' AND scope='%s' AND key='%s';`,
			charID, scope, key, value, description,
			charID, scope, key,
		)

		if err := database.Exec(sql, nil); err != nil {
			slog.Error("flagengine: upsert world flag failed",
				"char_id", charID, "scope", scope, "key", key, "err", err)
		}
	}
}

// LoadRelevantFlags carica i flag rilevanti per il turno corrente.
// Priorità: world, kingdom:*, faction:*, city:currentLocation, dungeon:currentDungeon, npc (ultimi 8).
// Massimo 30 flag totali.
func LoadRelevantFlags(database db.DBClient, charID, location, dungeonName string) []models.WorldFlag {
	cityScope := "city:" + normalizeScope(location)
	dungeonScope := "dungeon:" + normalizeScope(dungeonName)

	// Costruisce la query per caricare flag rilevanti con priorità.
	// Ordine: world, kingdom:*, faction:*, city:corrente, dungeon:corrente, npc:* (limite 8).
	var allFlags []models.WorldFlag

	// 1. Flag globali e di regno/fazione (non dipendono dalla posizione)
	sql := fmt.Sprintf(
		`SELECT * FROM world_flags WHERE character_id='%s' AND (scope='world' OR string::starts_with(scope, 'kingdom:') OR string::starts_with(scope, 'faction:')) ORDER BY updated_at DESC LIMIT 20;`,
		charID,
	)
	if results, err := database.Query(sql, nil); err == nil && len(results) > 0 {
		var flags []models.WorldFlag
		if err := results[0].All(&flags); err == nil {
			allFlags = append(allFlags, flags...)
		}
	}

	// 2. Flag città corrente
	if location != "" {
		sql = fmt.Sprintf(
			`SELECT * FROM world_flags WHERE character_id='%s' AND scope='%s' ORDER BY updated_at DESC LIMIT 10;`,
			charID, cityScope,
		)
		if results, err := database.Query(sql, nil); err == nil && len(results) > 0 {
			var flags []models.WorldFlag
			if err := results[0].All(&flags); err == nil {
				allFlags = append(allFlags, flags...)
			}
		}
	}

	// 3. Flag dungeon corrente (se in dungeon)
	if dungeonName != "" {
		sql = fmt.Sprintf(
			`SELECT * FROM world_flags WHERE character_id='%s' AND scope='%s' ORDER BY updated_at DESC LIMIT 10;`,
			charID, dungeonScope,
		)
		if results, err := database.Query(sql, nil); err == nil && len(results) > 0 {
			var flags []models.WorldFlag
			if err := results[0].All(&flags); err == nil {
				allFlags = append(allFlags, flags...)
			}
		}
	}

	// 4. Flag NPC (ultimi 8 aggiornati)
	sql = fmt.Sprintf(
		`SELECT * FROM world_flags WHERE character_id='%s' AND string::starts_with(scope, 'npc:') ORDER BY updated_at DESC LIMIT 8;`,
		charID,
	)
	if results, err := database.Query(sql, nil); err == nil && len(results) > 0 {
		var flags []models.WorldFlag
		if err := results[0].All(&flags); err == nil {
			allFlags = append(allFlags, flags...)
		}
	}

	// 5. Flag player — storia personale del personaggio (sempre caricati, senza filtro location)
	sql = fmt.Sprintf(
		`SELECT * FROM world_flags WHERE character_id='%s' AND scope='player' ORDER BY updated_at DESC;`,
		charID,
	)
	if results, err := database.Query(sql, nil); err == nil && len(results) > 0 {
		var flags []models.WorldFlag
		if err := results[0].All(&flags); err == nil {
			allFlags = append(allFlags, flags...)
		}
	}

	// Limite globale a 35 flag
	if len(allFlags) > 35 {
		allFlags = allFlags[:35]
	}

	return allFlags
}

// FormatFlagsForPrompt formatta i flag come blocco di testo per il prompt del GM.
// Restituisce stringa vuota se non ci sono flag.
func FormatFlagsForPrompt(flags []models.WorldFlag) string {
	if len(flags) == 0 {
		return ""
	}

	// Raggruppa per scope
	grouped := make(map[string][]models.WorldFlag)
	var scopeOrder []string
	for _, f := range flags {
		if _, exists := grouped[f.Scope]; !exists {
			scopeOrder = append(scopeOrder, f.Scope)
		}
		grouped[f.Scope] = append(grouped[f.Scope], f)
	}

	var sb strings.Builder
	sb.WriteString("## WORLD FLAGS — STATO PERSISTENTE DEL MONDO\n\n")

	for _, scope := range scopeOrder {
		items := grouped[scope]
		sb.WriteString(fmt.Sprintf("**%s**:\n", formatScopeLabel(scope)))
		for _, f := range items {
			if f.Description != "" {
				sb.WriteString(fmt.Sprintf("  • %s = %s (%s)\n", f.Key, f.Value, f.Description))
			} else {
				sb.WriteString(fmt.Sprintf("  • %s = %s\n", f.Key, f.Value))
			}
		}
	}
	sb.WriteString("\n")

	return sb.String()
}

// normalizeScope converte scope e chiavi in lowercase con underscore al posto degli spazi.
func normalizeScope(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

// formatScopeLabel restituisce un'etichetta leggibile per lo scope.
func formatScopeLabel(scope string) string {
	i := strings.IndexByte(scope, ':')
	if i == -1 {
		// scope semplice come "world"
		return strings.ToUpper(scope[:1]) + scope[1:]
	}
	scopeType := scope[:i]
	scopeName := strings.ReplaceAll(scope[i+1:], "_", " ")
	label := strings.ToUpper(scopeType[:1]) + scopeType[1:]
	return fmt.Sprintf("%s: %s", label, scopeName)
}
