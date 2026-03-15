package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// Claims represents the JWT payload.
type Claims struct {
	UserID string `json:"sub"`
	jwt.RegisteredClaims
}

// JWTManager handles token creation and validation.
type JWTManager struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTManager creates a new JWTManager. If secret is empty, a random one is generated
// (tokens will not survive server restarts).
func NewJWTManager(secret string, ttl time.Duration) (*JWTManager, error) {
	key := []byte(secret)
	if len(key) == 0 {
		key = make([]byte, 32)
		n, err := rand.Read(key)
		if err != nil {
			return nil, fmt.Errorf("failed to generate random JWT secret: %w", err)
		}
		if n != len(key) {
			return nil, fmt.Errorf("short read from crypto/rand: got %d bytes, want %d", n, len(key))
		}
	}
	return &JWTManager{secret: key, ttl: ttl}, nil
}

// CreateToken generates a signed JWT for the given user ID.
func (j *JWTManager) CreateToken(ctx context.Context, userID string) (string, error) {
	slog.DebugContext(ctx, "creating JWT token", slog.String(otelkeys.UserID, userID))
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(j.secret)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}
	slog.DebugContext(ctx, "JWT token created", slog.String(otelkeys.UserID, userID), slog.Time("expires_at", now.Add(j.ttl)))
	return signed, nil
}

// ValidateToken parses and validates a JWT, returning the claims if valid.
func (j *JWTManager) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	slog.DebugContext(ctx, "validating JWT token")
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			slog.DebugContext(ctx, "JWT token expired")
			return nil, ErrExpiredToken
		}
		slog.DebugContext(ctx, "JWT token invalid", slog.String(otelkeys.Error, err.Error()))
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		slog.DebugContext(ctx, "JWT token claims invalid")
		return nil, ErrInvalidToken
	}

	slog.DebugContext(ctx, "JWT token validated", slog.String(otelkeys.UserID, claims.UserID))
	return claims, nil
}
