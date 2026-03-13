package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

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
func (j *JWTManager) CreateToken(userID string) (string, error) {
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
	return token.SignedString(j.secret)
}

// ValidateToken parses and validates a JWT, returning the claims if valid.
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.secret, nil
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

	return claims, nil
}
