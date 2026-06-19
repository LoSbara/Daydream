package queue

import (
	"context"
	"fmt"
	"daydream/internal/models"
	"sync"
)

const maxQueueDepth = 5

// TurnJob rappresenta un turno da processare.
type TurnJob struct {
	UserID  string
	CharID  string
	Input   string
	TokenCh chan models.SSEEvent // il worker scrive qui, l'handler HTTP legge
}

// TurnProcessor è la funzione che gestisce un turno di gioco.
// Implementata dal game engine, iniettata qui per evitare import circolari.
type TurnProcessor func(ctx context.Context, job TurnJob)

// PlayerQueue gestisce una goroutine FIFO per ogni giocatore.
// Garantisce che i turni di un singolo giocatore vengano processati in ordine,
// anche se il client invia messaggi rapidamente.
type PlayerQueue struct {
	mu        sync.Mutex
	queues    map[string]chan TurnJob // keyed by userID
	processor TurnProcessor
}

func New(processor TurnProcessor) *PlayerQueue {
	return &PlayerQueue{
		queues:    make(map[string]chan TurnJob),
		processor: processor,
	}
}

// Enqueue aggiunge un turno alla coda del giocatore e restituisce il TurnJob
// (con TokenCh già allocato). Il chiamante deve leggere da TokenCh per
// ricevere i token SSE.
func (pq *PlayerQueue) Enqueue(ctx context.Context, userID, charID, input string) (TurnJob, error) {
	pq.mu.Lock()
	ch, ok := pq.queues[userID]
	if !ok {
		ch = make(chan TurnJob, maxQueueDepth)
		pq.queues[userID] = ch
		go pq.worker(userID, ch)
	}
	pq.mu.Unlock()

	if len(ch) >= maxQueueDepth {
		return TurnJob{}, fmt.Errorf("troppi turni in coda, riprova tra qualche secondo")
	}

	job := TurnJob{
		UserID:  userID,
		CharID:  charID,
		Input:   input,
		TokenCh: make(chan models.SSEEvent, 128),
	}

	select {
	case ch <- job:
		return job, nil
	case <-ctx.Done():
		return TurnJob{}, ctx.Err()
	}
}

// worker è la goroutine FIFO per un singolo giocatore.
// Non si chiude mai: se rimane idle troppo a lungo puoi aggiungere un ticker
// per la cleanup (Phase 2).
func (pq *PlayerQueue) worker(userID string, ch <-chan TurnJob) {
	for job := range ch {
		// Context non-cancellable per il turno: una volta iniziato va completato.
		// Il context dell'HTTP request è già scaduto (l'utente ha il suo canale).
		pq.processor(context.Background(), job)
	}
}
