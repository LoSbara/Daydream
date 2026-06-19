// MCP Tools Server — Daydream
//
// Server MCP via stdio (JSON-RPC 2.0).
// Espone 6 tool al GM Agent e agli altri agenti Claude.
//
// Avvio: go run ./cmd/mcp-server
// Oppure compilato: ./mcp-server (nella stessa dir del .env del backend)
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"daydream/internal/db"
	"daydream/internal/llm"
)

// ── JSON-RPC 2.0 types ───────────────────────────────────────────────────────

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ── MCP protocol types ───────────────────────────────────────────────────────

type toolDef struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema toolSchema `json:"inputSchema"`
}

type toolSchema struct {
	Type       string              `json:"type"`
	Properties map[string]schemaProp `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type schemaProp struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type toolResult struct {
	Content []contentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type contentItem struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// ── Server state ─────────────────────────────────────────────────────────────

type server struct {
	db  *db.Client // nil se SurrealDB non disponibile
	llm llm.Provider
	out *json.Encoder
}

// ── main ─────────────────────────────────────────────────────────────────────

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[mcp] nessun .env trovato, uso variabili di sistema")
	}

	// Connessione SurrealDB (opzionale — i tool che la richiedono degradano gracefully)
	var dbClient *db.Client
	if client, err := db.New(); err == nil {
		dbClient = client
		log.Println("[mcp] SurrealDB connesso")
	} else {
		log.Printf("[mcp] SurrealDB non disponibile (%v) — tool KB/state disabilitati", err)
	}

	// LLM provider per generate_npc_description
	var llmProvider llm.Provider
	if key := os.Getenv("LLM_API_KEY"); key != "" {
		switch os.Getenv("LLM_PROVIDER") {
		case "anthropic":
			model := os.Getenv("LLM_MODEL")
			if model == "" {
				model = "claude-sonnet-4-6"
			}
			llmProvider = llm.NewAnthropic(key, model)
		default:
			llmProvider = llm.NewOpenAI(
				os.Getenv("LLM_BASE_URL"),
				key,
				os.Getenv("LLM_MODEL"),
			)
		}
		log.Printf("[mcp] LLM provider pronto: %s", llmProvider.Name())
	}

	s := &server{
		db:  dbClient,
		llm: llmProvider,
		out: json.NewEncoder(os.Stdout),
	}

	log.Println("[mcp] Daydream MCP Tools Server avviato — in ascolto su stdio")
	s.serve(os.Stdin)
}

// serve legge richieste JSON-RPC da r (stdin) e scrive risposte su stdout.
func (s *server) serve(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1 MB buffer

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.writeError(nil, -32700, "Parse error: "+err.Error())
			continue
		}

		s.dispatch(req)
	}
}

func (s *server) dispatch(req rpcRequest) {
	// Notifiche (nessun id) — nessuna risposta richiesta
	if req.ID == nil && req.Method != "initialize" {
		return
	}

	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolCall(req)
	default:
		s.writeError(req.ID, -32601, "Method not found: "+req.Method)
	}
}

// ── MCP handshake ─────────────────────────────────────────────────────────────

func (s *server) handleInitialize(req rpcRequest) {
	s.writeResult(req.ID, map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "daydream-tools",
			"version": "1.0.0",
		},
	})
}

// ── tools/list ───────────────────────────────────────────────────────────────

func (s *server) handleToolsList(req rpcRequest) {
	tools := []toolDef{
		{
			Name:        "query_knowledge_base",
			Description: "Cerca nella knowledge base del mondo e del personaggio usando full-text search. Usa per recuperare lore di zone, dettagli NPC, eventi passati, assiomi del mondo.",
			InputSchema: toolSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"query":  {Type: "string", Description: "Cosa stai cercando (parole chiave o frase)"},
					"limit":  {Type: "number", Description: "Numero massimo di risultati (default 5, max 10)"},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "get_game_state_summary",
			Description: "Recupera un sommario leggibile dello stato corrente di un personaggio. Read-only.",
			InputSchema: toolSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"character_id": {Type: "string", Description: "ID del personaggio (record SurrealDB, es. character:abc123)"},
				},
				Required: []string{"character_id"},
			},
		},
		{
			Name:        "search_web",
			Description: "Cerca sul web per trovare ispirazione narrativa, dettagli di ambientazione fantasy, riferimenti culturali. NON usare per dati personali sul player.",
			InputSchema: toolSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"query": {Type: "string", Description: "Termine di ricerca"},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "roll_dice",
			Description: "Tira dadi con seed pseudo-deterministico. Per decisioni narrative che non sono meccanicamente risolte dai battle tags.",
			InputSchema: toolSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"notation": {Type: "string", Description: "Notazione dadi, es. '2d6+3', '1d20', 'd100'"},
					"purpose":  {Type: "string", Description: "Scopo del tiro (per logging narrativo)"},
				},
				Required: []string{"notation", "purpose"},
			},
		},
		{
			Name:        "generate_npc_description",
			Description: "Genera una descrizione dettagliata per un NPC nuovo con personalità, aspetto e motivazioni.",
			InputSchema: toolSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"npc_name":         {Type: "string", Description: "Nome dell'NPC"},
					"npc_type":         {Type: "string", Description: "Tipo (fabbro, mercante, antagonista, guardia, saggio...)"},
					"faction":          {Type: "string", Description: "Fazione di appartenenza (opzionale)"},
					"zone":             {Type: "string", Description: "Zona dove si trova l'NPC"},
					"personality_hint": {Type: "string", Description: "Suggerimento sulla personalità (opzionale)"},
				},
				Required: []string{"npc_name", "npc_type", "zone"},
			},
		},
		{
			Name:        "validate_json_response",
			Description: "Valida un draft di risposta GM contro lo schema obbligatorio prima di finalizzarla. Ritorna un report di problemi trovati.",
			InputSchema: toolSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"response_json": {Type: "string", Description: "JSON della risposta GM da validare (come stringa)"},
				},
				Required: []string{"response_json"},
			},
		},
	}

	s.writeResult(req.ID, map[string]any{"tools": tools})
}

// ── tools/call ────────────────────────────────────────────────────────────────

func (s *server) handleToolCall(req rpcRequest) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.writeError(req.ID, -32602, "Invalid params: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var result toolResult
	var toolErr error

	switch params.Name {
	case "query_knowledge_base":
		result, toolErr = s.toolQueryKB(ctx, params.Arguments)
	case "get_game_state_summary":
		result, toolErr = s.toolGameState(ctx, params.Arguments)
	case "search_web":
		result, toolErr = s.toolSearchWeb(ctx, params.Arguments)
	case "roll_dice":
		result, toolErr = s.toolRollDice(params.Arguments)
	case "generate_npc_description":
		result, toolErr = s.toolGenerateNPC(ctx, params.Arguments)
	case "validate_json_response":
		result, toolErr = s.toolValidateJSON(params.Arguments)
	default:
		s.writeError(req.ID, -32601, "Tool not found: "+params.Name)
		return
	}

	if toolErr != nil {
		s.writeResult(req.ID, toolResult{
			Content: []contentItem{{Type: "text", Text: "Errore: " + toolErr.Error()}},
			IsError: true,
		})
		return
	}

	s.writeResult(req.ID, result)
}

// ── Tool: query_knowledge_base ────────────────────────────────────────────────

func (s *server) toolQueryKB(ctx context.Context, args map[string]any) (toolResult, error) {
	if s.db == nil {
		return toolResult{}, fmt.Errorf("SurrealDB non disponibile")
	}
	query, _ := args["query"].(string)
	if query == "" {
		return toolResult{}, fmt.Errorf("parametro 'query' obbligatorio")
	}
	limit := 5
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 10 {
			limit = 10
		}
	}

	// BM25 full-text search sulla knowledge_base
	sql := fmt.Sprintf(`
		SELECT title, content, category
		FROM knowledge_base
		WHERE content @@ $q OR title @@ $q
		LIMIT %d`, limit)

	qr, err := s.db.Query(sql, map[string]any{"q": query})
	if err != nil {
		return toolResult{}, fmt.Errorf("query KB: %w", err)
	}

	type kbEntry struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		Category string `json:"category"`
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Risultati per \"%s\":\n\n", query))
	found := 0

	for _, r := range qr {
		var entries []kbEntry
		if err := r.All(&entries); err != nil {
			continue
		}
		for _, e := range entries {
			sb.WriteString(fmt.Sprintf("### %s [%s]\n%s\n\n", e.Title, e.Category, e.Content))
			found++
		}
	}

	if found == 0 {
		sb.WriteString("Nessun documento trovato nella Knowledge Base per questa query.")
	}

	return toolResult{Content: []contentItem{{Type: "text", Text: sb.String()}}}, nil
}

// ── Tool: get_game_state_summary ─────────────────────────────────────────────

func (s *server) toolGameState(ctx context.Context, args map[string]any) (toolResult, error) {
	if s.db == nil {
		return toolResult{}, fmt.Errorf("SurrealDB non disponibile")
	}
	charID, _ := args["character_id"].(string)
	if charID == "" {
		return toolResult{}, fmt.Errorf("parametro 'character_id' obbligatorio")
	}

	// Character
	charQR, err := s.db.QueryOne(
		"SELECT name, job, level, experience, money, stats, status_effects FROM $id",
		map[string]any{"id": charID},
	)
	if err != nil {
		return toolResult{}, fmt.Errorf("query personaggio: %w", err)
	}
	var charData map[string]any
	if err := charQR.First(&charData); err != nil {
		return toolResult{}, fmt.Errorf("personaggio non trovato: %w", err)
	}

	// Game session
	sessQR, err := s.db.QueryOne(
		"SELECT location, sub_location, game_state, combat_active, context_memo, quests_active FROM game_session WHERE character_id = $cid",
		map[string]any{"cid": charID},
	)
	if err != nil {
		return toolResult{}, fmt.Errorf("query sessione: %w", err)
	}
	var sessData map[string]any
	if err := sessQR.First(&sessData); err != nil {
		return toolResult{}, fmt.Errorf("sessione non trovata: %w", err)
	}

	summary := formatStateSummary(charData, sessData)
	return toolResult{Content: []contentItem{{Type: "text", Text: summary}}}, nil
}

func formatStateSummary(char, sess map[string]any) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Stato Personaggio\n"))
	sb.WriteString(fmt.Sprintf("- Nome: %v | Classe: %v | Livello: %v\n", char["name"], char["job"], char["level"]))
	sb.WriteString(fmt.Sprintf("- EXP: %v | Gold: %v\n", char["experience"], char["money"]))

	if stats, ok := char["stats"].(map[string]any); ok {
		if hp, ok := stats["HP"].(map[string]any); ok {
			sb.WriteString(fmt.Sprintf("- HP: %v/%v", hp["current"], hp["max"]))
		}
		if mp, ok := stats["MP"].(map[string]any); ok {
			sb.WriteString(fmt.Sprintf(" | MP: %v/%v", mp["current"], mp["max"]))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("\n## Sessione\n"))
	sb.WriteString(fmt.Sprintf("- Posizione: %v", sess["location"]))
	if sub, ok := sess["sub_location"].(string); ok && sub != "" {
		sb.WriteString(fmt.Sprintf(" — %v", sub))
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("- Stato: %v | Combattimento: %v\n", sess["game_state"], sess["combat_active"]))

	if memo, ok := sess["context_memo"].(string); ok && memo != "" {
		if len(memo) > 300 {
			memo = memo[:300] + "…"
		}
		sb.WriteString(fmt.Sprintf("\n## Memo Scena\n%s\n", memo))
	}

	return sb.String()
}

// ── Tool: search_web ─────────────────────────────────────────────────────────

func (s *server) toolSearchWeb(ctx context.Context, args map[string]any) (toolResult, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return toolResult{}, fmt.Errorf("parametro 'query' obbligatorio")
	}

	// DuckDuckGo Instant Answer API (nessuna chiave richiesta)
	apiURL := "https://api.duckduckgo.com/?q=" + url.QueryEscape(query) + "&format=json&no_html=1&skip_disambig=1"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return toolResult{}, err
	}
	req.Header.Set("User-Agent", "Daydream-MCP/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return toolResult{}, fmt.Errorf("HTTP: %w", err)
	}
	defer resp.Body.Close()

	var ddg struct {
		AbstractText string `json:"AbstractText"`
		AbstractURL  string `json:"AbstractURL"`
		Answer       string `json:"Answer"`
		RelatedTopics []struct {
			Text string `json:"Text"`
		} `json:"RelatedTopics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ddg); err != nil {
		return toolResult{}, fmt.Errorf("parse risposta DuckDuckGo: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Risultati web per \"%s\":\n\n", query))

	if ddg.Answer != "" {
		sb.WriteString(fmt.Sprintf("**Risposta diretta**: %s\n\n", ddg.Answer))
	}
	if ddg.AbstractText != "" {
		sb.WriteString(fmt.Sprintf("**Sintesi**: %s\n", ddg.AbstractText))
		if ddg.AbstractURL != "" {
			sb.WriteString(fmt.Sprintf("Fonte: %s\n", ddg.AbstractURL))
		}
		sb.WriteString("\n")
	}

	limit := 3
	for i, rel := range ddg.RelatedTopics {
		if i >= limit || rel.Text == "" {
			break
		}
		sb.WriteString(fmt.Sprintf("- %s\n", rel.Text))
	}

	if sb.Len() < 60 {
		sb.WriteString("Nessun risultato significativo trovato dalla Instant Answer API. Prova una query più specifica.")
	}

	return toolResult{Content: []contentItem{{Type: "text", Text: sb.String()}}}, nil
}

// ── Tool: roll_dice ──────────────────────────────────────────────────────────

var diceRe = regexp.MustCompile(`^(\d*)d(\d+)([+-]\d+)?$`)

func (s *server) toolRollDice(args map[string]any) (toolResult, error) {
	notation, _ := args["notation"].(string)
	purpose, _ := args["purpose"].(string)
	if notation == "" {
		return toolResult{}, fmt.Errorf("parametro 'notation' obbligatorio")
	}

	notation = strings.ToLower(strings.TrimSpace(notation))
	m := diceRe.FindStringSubmatch(notation)
	if m == nil {
		return toolResult{}, fmt.Errorf("notazione dadi non valida: %q (usa es. '2d6+3', '1d20', 'd100')", notation)
	}

	numDice := 1
	if m[1] != "" {
		numDice, _ = strconv.Atoi(m[1])
	}
	sides, _ := strconv.Atoi(m[2])
	modifier := 0
	if m[3] != "" {
		modifier, _ = strconv.Atoi(m[3])
	}

	if sides < 2 || numDice < 1 || numDice > 20 {
		return toolResult{}, fmt.Errorf("parametri dado fuori range: %dd%d", numDice, sides)
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rolls := make([]int, numDice)
	total := 0
	for i := range rolls {
		rolls[i] = rng.Intn(sides) + 1
		total += rolls[i]
	}
	total += modifier

	rollStrs := make([]string, len(rolls))
	for i, r := range rolls {
		rollStrs[i] = strconv.Itoa(r)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🎲 **%s** → %s\n", notation, purpose))
	sb.WriteString(fmt.Sprintf("Risultati: [%s]", strings.Join(rollStrs, ", ")))
	if modifier != 0 {
		sb.WriteString(fmt.Sprintf(" %+d", modifier))
	}
	sb.WriteString(fmt.Sprintf("\n**Totale: %d**", total))

	return toolResult{Content: []contentItem{{Type: "text", Text: sb.String()}}}, nil
}

// ── Tool: generate_npc_description ───────────────────────────────────────────

func (s *server) toolGenerateNPC(ctx context.Context, args map[string]any) (toolResult, error) {
	if s.llm == nil {
		return toolResult{}, fmt.Errorf("LLM provider non configurato")
	}
	name, _ := args["npc_name"].(string)
	npcType, _ := args["npc_type"].(string)
	zone, _ := args["zone"].(string)
	faction, _ := args["faction"].(string)
	hint, _ := args["personality_hint"].(string)

	if name == "" || npcType == "" || zone == "" {
		return toolResult{}, fmt.Errorf("npc_name, npc_type e zone sono obbligatori")
	}

	contextParts := []string{fmt.Sprintf("Tipo: %s", npcType), fmt.Sprintf("Zona: %s", zone)}
	if faction != "" {
		contextParts = append(contextParts, fmt.Sprintf("Fazione: %s", faction))
	}
	if hint != "" {
		contextParts = append(contextParts, fmt.Sprintf("Personalità: %s", hint))
	}

	msgs := []llm.Message{
		{
			Role: "system",
			Content: `Sei un character designer per un VRMMO dark fantasy chiamato Daydream.
Crea descrizioni NPC dettagliate e memorabili.
Formato output: aspetto fisico (2-3 frasi), personalità e motivazioni (2-3 frasi), ruolo nel mondo e segreti (2-3 frasi).
Tono dark fantasy, italiano. Massimo 200 parole totali.`,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Crea la descrizione di: **%s**\n%s", name, strings.Join(contextParts, " | ")),
		},
	}

	result, err := s.llm.Complete(ctx, msgs, llm.CompleteOpts{MaxTokens: 400, Temperature: 0.8})
	if err != nil {
		return toolResult{}, fmt.Errorf("generazione LLM: %w", err)
	}

	output := fmt.Sprintf("## %s\n\n%s", name, strings.TrimSpace(result))
	return toolResult{Content: []contentItem{{Type: "text", Text: output}}}, nil
}

// ── Tool: validate_json_response ─────────────────────────────────────────────

func (s *server) toolValidateJSON(args map[string]any) (toolResult, error) {
	raw, _ := args["response_json"].(string)
	if raw == "" {
		return toolResult{}, fmt.Errorf("parametro 'response_json' obbligatorio")
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return toolResult{Content: []contentItem{{Type: "text",
			Text: fmt.Sprintf("❌ JSON NON VALIDO: %v\n\nCorreggi la sintassi prima di inviare.", err),
		}}, IsError: true}, nil
	}

	var issues []string
	var ok []string

	// narrative (obbligatorio, stringa non vuota)
	if narr, exists := resp["narrative"]; !exists {
		issues = append(issues, "❌ 'narrative' mancante (campo OBBLIGATORIO)")
	} else if s, _ := narr.(string); s == "" {
		issues = append(issues, "❌ 'narrative' è vuoto")
	} else {
		length := len(s)
		if length < 80 {
			issues = append(issues, fmt.Sprintf("⚠️  'narrative' troppo corto (%d chars, minimo 80)", length))
		} else if length > 3000 {
			issues = append(issues, fmt.Sprintf("⚠️  'narrative' molto lungo (%d chars)", length))
		} else {
			ok = append(ok, fmt.Sprintf("✅ narrative OK (%d chars)", length))
		}
	}

	// battle_tags in combattimento
	if combat, hasCombat := resp["state_updates"]; hasCombat {
		if su, ok2 := combat.(map[string]any); ok2 {
			if gs, ok3 := su["game_state"].(map[string]any); ok3 {
				if ca, _ := gs["combat_active"].(bool); ca {
					if tags, hasTags := resp["battle_tags"]; !hasTags || tags == nil {
						issues = append(issues, "❌ combat_active=true ma 'battle_tags' assente")
					}
				}
			}
		}
	}

	// Nessun gold in state_updates.player
	if su, ok2 := resp["state_updates"].(map[string]any); ok2 {
		if player, ok3 := su["player"].(map[string]any); ok3 {
			if _, hasGold := player["money"]; hasGold {
				issues = append(issues, "❌ VIOLAZIONE: 'money' in state_updates.player — usa GOLD_GAIN/GOLD_LOSE nei battle_tags")
			}
		}
	}

	// context_memo
	if _, hasMemo := resp["context_memo"]; hasMemo {
		ok = append(ok, "✅ context_memo presente")
	} else {
		issues = append(issues, "⚠️  'context_memo' assente (consigliato aggiornarlo ogni turno)")
	}

	// action_category
	if cat, _ := resp["action_category"].(string); cat != "" {
		validCats := map[string]bool{"conversation": true, "combat": true, "exploration": true,
			"travel_local": true, "travel_regional": true, "travel_long": true, "rest": true, "crafting": true}
		if validCats[cat] {
			ok = append(ok, fmt.Sprintf("✅ action_category: %s", cat))
		} else {
			issues = append(issues, fmt.Sprintf("⚠️  action_category '%s' non riconosciuta", cat))
		}
	}

	var sb strings.Builder
	score := 100 - len(issues)*20
	if score < 0 {
		score = 0
	}
	sb.WriteString(fmt.Sprintf("## Validazione GM Response — Score: %d/100\n\n", score))

	if len(ok) > 0 {
		sb.WriteString("### Checks OK\n")
		for _, o := range ok {
			sb.WriteString(o + "\n")
		}
		sb.WriteString("\n")
	}
	if len(issues) > 0 {
		sb.WriteString("### Problemi trovati\n")
		for _, iss := range issues {
			sb.WriteString(iss + "\n")
		}
	} else {
		sb.WriteString("### Nessun problema critico trovato ✅\n")
	}

	return toolResult{Content: []contentItem{{Type: "text", Text: sb.String()}}}, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func (s *server) writeResult(id any, result any) {
	resp := rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
	if err := s.out.Encode(resp); err != nil {
		log.Printf("[mcp] errore write response: %v", err)
	}
}

func (s *server) writeError(id any, code int, message string) {
	resp := rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message},
	}
	if err := s.out.Encode(resp); err != nil {
		log.Printf("[mcp] errore write error response: %v", err)
	}
}
