package embedding

import "context"

// NoopProvider è un provider che restituisce vettori zero.
// Usato quando il RAG è disabilitato (EMBED_ENABLED=false).
type NoopProvider struct{ dims int }

func NewNoop(dimensions int) *NoopProvider {
	if dimensions == 0 {
		dimensions = 768
	}
	return &NoopProvider{dims: dimensions}
}

func (p *NoopProvider) Name() string    { return "noop" }
func (p *NoopProvider) Dimensions() int { return p.dims }
func (p *NoopProvider) Embed(_ context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range result {
		result[i] = make([]float32, p.dims)
	}
	return result, nil
}
