package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"daydream/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Claims struct {
	jwt.RegisteredClaims
	TokenType string `json:"typ"` // "access" | "refresh"
}

var (
	ErrInvalidToken = errors.New("token non valido")
	ErrExpiredToken = errors.New("token scaduto")
)

// GenerateTokenPair genera access + refresh token per un utente.
func GenerateTokenPair(userID string) (TokenPair, error) {
	access, err := generateToken(userID, "access", accessExpiry())
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := generateToken(userID, "refresh", refreshExpiry())
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

// GenerateAccessToken genera solo un access token (usato nel refresh flow).
func GenerateAccessToken(userID string) (string, error) {
	return generateToken(userID, "access", accessExpiry())
}

// ValidateAccessToken valida un access token e ritorna le claims.
func ValidateAccessToken(tokenStr string) (*Claims, error) {
	return parseToken(tokenStr, accessSecret(), "access")
}

// ValidateRefreshToken valida un refresh token e ritorna le claims.
func ValidateRefreshToken(tokenStr string) (*Claims, error) {
	return parseToken(tokenStr, refreshSecret(), "refresh")
}

func generateToken(userID, tokenType string, expiry time.Duration) (string, error) {
	secret := accessSecret()
	if tokenType == "refresh" {
		secret = refreshSecret()
	}

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			Issuer:    config.AppName,
		},
		TokenType: tokenType,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("jwt sign: %w", err)
	}
	return signed, nil
}

func parseToken(tokenStr, secret, expectedType string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("algoritmo inatteso: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.TokenType != expectedType {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func accessSecret() string  { return os.Getenv("JWT_ACCESS_SECRET") }
func refreshSecret() string { return os.Getenv("JWT_REFRESH_SECRET") }

func accessExpiry() time.Duration {
	d, err := time.ParseDuration(os.Getenv("JWT_ACCESS_EXPIRY"))
	if err != nil {
		return 15 * time.Minute
	}
	return d
}

func refreshExpiry() time.Duration {
	d, err := time.ParseDuration(os.Getenv("JWT_REFRESH_EXPIRY"))
	if err != nil {
		return 168 * time.Hour
	}
	return d
}
