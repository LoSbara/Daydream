package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OpenAICompatProvider usa l'API di embedding OpenAI-compatible.
// Funziona con OpenAI (text-embedding-3-small), DeepSeek e qualsiasi
// provider con lo stesso formato: POST /v1/embeddings.
type OpenAICompatProvider struct {
	baseURL    string
	apiKey     string
	model      string
	dimensions int
	client     *http.Client
}

func NewOpenAICompat(baseURL, apiKey, model string, dimensions int) *OpenAICompatProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "text-embedding-3-small"
	}
	if dimensions == 0 {
		dimensions = 1536
	}
	return &OpenAICompatProvider{
		baseURL:    baseURL,
		apiKey:     apiKey,
		model:      model,
		dimensions: dimensions,
		client:     &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *OpenAICompatProvider) Name() string       { return fmt.Sprintf("openai-compat/%s", p.model) }
func (p *OpenAICompatProvider) Dimensions() int    { return p.dimensions }

func (p *OpenAICompatProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(map[string]any{
		"model": p.model,
		"input": texts,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai embed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai embed: status %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("openai embed decode: %w", err)
	}
	if len(result.Data) != len(texts) {
		return nil, fmt.Errorf("openai embed: expected %d vettori, ricevuti %d", len(texts), len(result.Data))
	}

	// Ordina per indice (l'API li restituisce in ordine ma per sicurezza)
	out := make([][]float32, len(texts))
	for _, d := range result.Data {
		if d.Index < len(out) {
			out[d.Index] = d.Embedding
		}
	}
	return out, nil
}
