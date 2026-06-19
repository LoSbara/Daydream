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

// OpenAIProvider implementa Provider per qualsiasi API OpenAI-compatible.
// Funziona con: OpenAI, DeepSeek, Groq, Together, Ollama, LM Studio, ecc.
type OpenAIProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func NewOpenAI(baseURL, apiKey, model string) *OpenAIProvider {
	return &OpenAIProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *OpenAIProvider) Name() string { return "openai-compat" }

// Complete esegue una chiamata non-streaming e restituisce il testo completo.
func (p *OpenAIProvider) Complete(ctx context.Context, messages []Message, opts CompleteOpts) (string, error) {
	model := p.model
	if opts.Model != "" {
		model = opts.Model
	}
	maxTokens := 2048
	if opts.MaxTokens > 0 {
		maxTokens = opts.MaxTokens
	}
	temp := 0.7
	if opts.Temperature > 0 {
		temp = opts.Temperature
	}

	payload := map[string]any{
		"model":       model,
		"messages":    messages,
		"max_tokens":  maxTokens,
		"temperature": temp,
		"stream":      false,
	}
	if opts.JSONMode {
		payload["response_format"] = map[string]string{"type": "json_object"}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM error %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("nessun contenuto nella risposta LLM")
	}

	return result.Choices[0].Message.Content, nil
}

// Stream esegue una chiamata streaming inviando i token su out.
// Legge il formato SSE di OpenAI: data: {...}\ndata: [DONE]
func (p *OpenAIProvider) Stream(ctx context.Context, messages []Message, opts CompleteOpts, out chan<- string) error {
	model := p.model
	if opts.Model != "" {
		model = opts.Model
	}
	maxTokens := 2048
	if opts.MaxTokens > 0 {
		maxTokens = opts.MaxTokens
	}
	temp := 0.7
	if opts.Temperature > 0 {
		temp = opts.Temperature
	}

	payload := map[string]any{
		"model":       model,
		"messages":    messages,
		"max_tokens":  maxTokens,
		"temperature": temp,
		"stream":      true,
	}
	if opts.JSONMode {
		payload["response_format"] = map[string]string{"type": "json_object"}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Timeout separato per streaming (senza deadline fissa, usiamo il context)
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	streamClient := &http.Client{} // nessun timeout fisso per lo streaming
	resp, err := streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("LLM stream error %d: %s", resp.StatusCode, string(raw))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// Linea vuota o heartbeat
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // ignora chunk malformati
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		text := chunk.Choices[0].Delta.Content
		if text != "" {
			select {
			case out <- text:
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if chunk.Choices[0].FinishReason != nil && *chunk.Choices[0].FinishReason != "" {
			break
		}
	}

	return scanner.Err()
}
