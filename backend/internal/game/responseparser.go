package game

import (
	"encoding/json"
	"fmt"
	"strings"

	"daydream/internal/models"
)

// narrativeExtractor è una FSM che estrae il valore del campo "narrative" da un JSON stream.
// Viene usata da callLLMStreaming per emettere token narrativi in real-time al client SSE.
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

// parseGMResponse tenta di deserializzare il JSON del GM.
// Pulisce il JSON da eventuali markdown code block (```json...```).
func parseGMResponse(raw string) (*models.GMResponse, error) {
	raw = strings.TrimSpace(raw)

	// Rimuovi markdown code block se presente
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) >= 2 {
			lines = lines[1:]
			if len(lines) > 0 && lines[len(lines)-1] == "```" {
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
