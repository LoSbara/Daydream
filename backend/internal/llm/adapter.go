package llm

import "context"

// Message è un singolo messaggio nella conversazione con l'LLM.
type Message struct {
	Role    string `json:"role"`    // "system" | "user" | "assistant"
	Content string `json:"content"`
}

// CompleteOpts contiene le opzioni per una chiamata all'LLM.
type CompleteOpts struct {
	Model       string
	MaxTokens   int
	Temperature float64
	JSONMode    bool   // richiede risposta in JSON valido
	Stream      bool   // abilita lo streaming
}

// Provider è l'interfaccia LLM-agnostica.
// Tutti i provider (OpenAI, Anthropic, Gemini, Ollama) la implementano.
type Provider interface {
	// Complete esegue una chiamata non-streaming e restituisce il testo completo.
	Complete(ctx context.Context, messages []Message, opts CompleteOpts) (string, error)

	// Stream esegue una chiamata streaming, inviando i token su out.
	// Chiude out quando la generazione è terminata o in caso di errore.
	Stream(ctx context.Context, messages []Message, opts CompleteOpts, out chan<- string) error

	// Name restituisce il nome del provider (per logging).
	Name() string
}
