package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"daydream/internal/auth"
)

// chatRateLimiter è un token bucket in-memory per l'endpoint /api/chat.
// Limite: 1 turno ogni 5 secondi per utente autenticato.
// Scegliamo 5s perché il turno GM impiega tipicamente 3-8s:
// un giocatore che aspetta la risposta non può premere invio più veloce.
type chatRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]time.Time // userID → momento dell'ultimo invio
}

var globalChatLimiter = &chatRateLimiter{
	buckets: make(map[string]time.Time),
}

// ChatRateLimit è il middleware Gin per il rate limiting del chat.
// Restituisce 429 con header Retry-After se il limite è superato.
func ChatRateLimit() gin.HandlerFunc {
	const minInterval = 5 * time.Second

	return func(c *gin.Context) {
		userID := auth.GetUserID(c)
		if userID == "" {
			c.Next()
			return
		}

		globalChatLimiter.mu.Lock()
		last, exists := globalChatLimiter.buckets[userID]
		now := time.Now()
		if exists && now.Sub(last) < minInterval {
			retryAfter := minInterval - now.Sub(last)
			globalChatLimiter.mu.Unlock()
			c.Header("Retry-After", retryAfter.String())
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "troppi messaggi ravvicinati",
				"retry_after": retryAfter.Seconds(),
			})
			c.Abort()
			return
		}
		globalChatLimiter.buckets[userID] = now

		// Pulizia periodica: rimuovi bucket scaduti (> 1 min inattivi)
		if len(globalChatLimiter.buckets) > 1000 {
			threshold := now.Add(-1 * time.Minute)
			for uid, t := range globalChatLimiter.buckets {
				if t.Before(threshold) {
					delete(globalChatLimiter.buckets, uid)
				}
			}
		}
		globalChatLimiter.mu.Unlock()

		c.Next()
	}
}
