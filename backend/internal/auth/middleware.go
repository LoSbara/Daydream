package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const ContextUserID = "user_id"

// Middleware estrae e valida il JWT Bearer dall'header Authorization.
// In caso di token valido, imposta c.Keys["user_id"] = subject del token.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token mancante"})
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := ValidateAccessToken(tokenStr)
		if err != nil {
			status := http.StatusUnauthorized
			msg := "token non valido"
			if err == ErrExpiredToken {
				msg = "token scaduto"
			}
			c.AbortWithStatusJSON(status, gin.H{"error": msg})
			return
		}

		c.Set(ContextUserID, claims.Subject)
		c.Next()
	}
}

// GetUserID è un helper per estrarre l'user ID dal context Gin.
func GetUserID(c *gin.Context) string {
	v, _ := c.Get(ContextUserID)
	id, _ := v.(string)
	return id
}
