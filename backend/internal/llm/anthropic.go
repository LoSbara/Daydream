package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const anthropicAPIVersion = "2023-06-01"
const anthropicBaseURL = "https://api.anthropic.com/v1"

// AnthropicProvider implementa Provider per l'API Anthropic Claude.
// Nota: Anthropic vuole system come campo top-level, non dentro messages.
type AnthropicProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewAnthropic(apiKey, model string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

// splitSystemMessages estrae il messaggio di sistema dalla lista.
// Anthropic lo vuole nel campo "system" top-level, non in messages.
func splitSystemMessages(messages []Message) (system string, filtered []Message) {
	for _, m := range messages {
		if m.Role == "system" {
			if system != "" {
				system += "\n\n" + m.Content
			} else {
				system = m.Content
			}
		} else {
			filtered = append(filtered, m)
		}
	}
	return
}

type anthropicRequest struct {
	Model     string           `json:"model"`
	System    string           `json:"system,omitempty"`
	Messages  []anthropicMsg   `json:"messages"`
	MaxTokens int              `json:"max_tokens"`
	Stream    bool             `json:"stream"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func toAnthropicMsgs(msgs []Message) []anthropicMsg {
	out := make([]anthropicMsg, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, anthropicMsg{Role: m.Role, Content: m.Content})
	}
	return out
}

func (p *AnthropicProvider) buildRequest(messages []Message, opts CompleteOpts, stream bool) (anthropicRequest, error) {
	model := p.model
	if opts.Model != "" {
		model = opts.Model
	}
	maxTokens := 2048
	if opts.MaxTokens > 0 {
		maxTokens = opts.MaxTokens
	}

	system, filtered := splitSystemMessages(messages)

	return anthropicRequest{
		Model:     model,
		System:    system,
		Messages:  toAnthropicMsgs(filtered),
		MaxTokens: maxTokens,
		Stream:    stream,
	}, nil
}

func (p *AnthropicProvider) doRequest(ctx context.Context, payload any) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicBaseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)

	return p.client.Do(req)
}

func (p *AnthropicProvider) Complete(ctx context.Context, messages []Message, opts CompleteOpts) (string, error) {
	payload, err := p.buildRequest(messages, opts, false)
	if err != nil {
		return "", err
	}

	resp, err := p.doRequest(ctx, payload)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("anthropic error %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	for _, block := range result.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}
	return "", fmt.Errorf("nessun blocco testo nella risposta Anthropic")
}

func (p *AnthropicProvider) Stream(ctx context.Context, messages []Message, opts CompleteOpts, out chan<- string) error {
	payload, err := p.buildRequest(messages, opts, true)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicBaseURL+"/messages", func() *bytes.Reader {
		b, _ := json.Marshal(payload)
		return bytes.NewReader(b)
	}())
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)

	streamClient := &http.Client{}
	resp, err := streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("anthropic stream error %d: %s", resp.StatusCode, string(raw))
	}

	// Anthropic SSE format:
	// data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"..."}}
	// data: {"type":"message_stop"}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" && event.Delta.Text != "" {
			select {
			case out <- event.Delta.Text:
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if event.Type == "message_stop" {
			break
		}
	}

	return scanner.Err()
}
