package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"daydream/internal/auth"
	"daydream/internal/db"
)

type User struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	CreatedAt string  `json:"created_at"`
	LastLogin *string `json:"last_login,omitempty"`
}

// POST /api/auth/register
func (h *Handler) Register(c *gin.Context) {
	var body struct {
		Username string `json:"username" binding:"required,min=3,max=32"`
		Password string `json:"password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "username (3-32 char) e password (min 8 char) obbligatori")
		return
	}

	body.Username = strings.TrimSpace(strings.ToLower(body.Username))

	// Verifica unicità username
	results, err := h.DB.Query(
		"SELECT id FROM user WHERE username = $username",
		map[string]any{"username": body.Username},
	)
	if err != nil {
		internalError(c, err)
		return
	}
	var existing []User
	if err := results[0].All(&existing); err == nil && len(existing) > 0 {
		conflict(c, "username già in uso")
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		internalError(c, err)
		return
	}

	// Crea utente
	results, err = h.DB.Query(
		`CREATE user CONTENT {
			username: $username,
			password_hash: $password_hash,
			created_at: time::now()
		}`,
		map[string]any{
			"username":      body.Username,
			"password_hash": string(hash),
		},
	)
	if err != nil {
		internalError(c, err)
		return
	}

	var user User
	if err := results[0].First(&user); err != nil {
		internalError(c, err)
		return
	}

	tokens, err := auth.GenerateTokenPair(user.ID)
	if err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"access_token":  tokens.AccessToken,
			"refresh_token": tokens.RefreshToken,
			"user":          user,
		},
	})
}

// POST /api/auth/login
func (h *Handler) Login(c *gin.Context) {
	var body struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "username e password obbligatori")
		return
	}

	body.Username = strings.TrimSpace(strings.ToLower(body.Username))

	results, err := h.DB.Query(
		"SELECT id, username, password_hash, created_at, last_login FROM user WHERE username = $username",
		map[string]any{"username": body.Username},
	)
	if err != nil {
		internalError(c, err)
		return
	}

	var raw struct {
		ID           string  `json:"id"`
		Username     string  `json:"username"`
		PasswordHash string  `json:"password_hash"`
		CreatedAt    string  `json:"created_at"`
		LastLogin    *string `json:"last_login,omitempty"`
	}
	if err := results[0].First(&raw); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			unauthorized(c, "credenziali non valide")
			return
		}
		internalError(c, err)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(raw.PasswordHash), []byte(body.Password)); err != nil {
		unauthorized(c, "credenziali non valide")
		return
	}

	// Aggiorna last_login (fire and forget, non blocca la risposta)
	go h.DB.Exec( //nolint
		"UPDATE $user_id SET last_login = time::now()",
		map[string]any{"user_id": raw.ID},
	)

	tokens, err := auth.GenerateTokenPair(raw.ID)
	if err != nil {
		internalError(c, err)
		return
	}

	ok(c, gin.H{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"user": User{
			ID:        raw.ID,
			Username:  raw.Username,
			CreatedAt: raw.CreatedAt,
			LastLogin: raw.LastLogin,
		},
	})
}

// POST /api/auth/refresh
func (h *Handler) Refresh(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "refresh_token obbligatorio")
		return
	}

	claims, err := auth.ValidateRefreshToken(body.RefreshToken)
	if err != nil {
		unauthorized(c, err.Error())
		return
	}

	accessToken, err := auth.GenerateAccessToken(claims.Subject)
	if err != nil {
		internalError(c, err)
		return
	}

	ok(c, gin.H{"access_token": accessToken})
}

// GET /api/auth/me  (richiede JWT)
func (h *Handler) Me(c *gin.Context) {
	userID := auth.GetUserID(c)

	results, err := h.DB.Query(
		"SELECT id, username, created_at, last_login FROM user WHERE id = type::record($user_id)",
		map[string]any{"user_id": userID},
	)
	if err != nil {
		internalError(c, err)
		return
	}

	var user User
	if err := results[0].First(&user); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			unauthorized(c, "utente non trovato")
			return
		}
		internalError(c, err)
		return
	}

	ok(c, user)
}
