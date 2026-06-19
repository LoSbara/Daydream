package embedding

import "context"

// Provider è l'interfaccia per i servizi di embedding testuale.
// Restituisce vettori float32 normalizzati pronti per la cosine similarity.
type Provider interface {
	// Embed calcola gli embedding per una lista di testi in batch.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	// Dimensions restituisce la dimensione del vettore prodotto.
	Dimensions() int
	// Name restituisce il nome del provider (per logging).
	Name() string
}
