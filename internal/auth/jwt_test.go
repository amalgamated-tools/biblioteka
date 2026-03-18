package auth

import (
	"errors"
	"testing"
	"time"
)

func TestNewJWTManager_WithSecret(t *testing.T) {
	jm, err := NewJWTManager("mysecret", time.Hour)
	if err != nil {
		failNowf(t, "NewJWTManager() unexpected error: %v", err)
	}
	if jm == nil {
		failNow(t, "NewJWTManager() returned nil")
	}
}

func TestNewJWTManager_RandomSecret(t *testing.T) {
	jm, err := NewJWTManager("", time.Hour)
	if err != nil {
		failNowf(t, "NewJWTManager() with empty secret unexpected error: %v", err)
	}
	if jm == nil {
		failNow(t, "NewJWTManager() returned nil")
	}
	if len(jm.secret) != 32 {
		failf(t, "expected 32-byte random secret, got %d bytes", len(jm.secret))
	}
}

func TestCreateAndValidateToken(t *testing.T) {
	jm, err := NewJWTManager("testsecret", time.Hour)
	if err != nil {
		failNowf(t, "NewJWTManager() error: %v", err)
	}

	userID := "user-123"
	token, err := jm.CreateToken(t.Context(), userID)
	if err != nil {
		failNowf(t, "CreateToken() error: %v", err)
	}
	if token == "" {
		failNow(t, "CreateToken() returned empty token")
	}

	claims, err := jm.ValidateToken(t.Context(), token)
	if err != nil {
		failNowf(t, "ValidateToken() error: %v", err)
	}
	if claims.UserID != userID {
		failf(t, "ValidateToken() UserID = %q, want %q", claims.UserID, userID)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	jm, err := NewJWTManager("testsecret", time.Hour)
	if err != nil {
		failNowf(t, "NewJWTManager() error: %v", err)
	}

	_, err = jm.ValidateToken(t.Context(), "not-a-valid-token")
	if !errors.Is(err, ErrInvalidToken) {
		failf(t, "ValidateToken() with invalid token: got %v, want ErrInvalidToken", err)
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Create manager with very short TTL
	jm, err := NewJWTManager("testsecret", -time.Second)
	if err != nil {
		failNowf(t, "NewJWTManager() error: %v", err)
	}

	token, err := jm.CreateToken(t.Context(), "user-123")
	if err != nil {
		failNowf(t, "CreateToken() error: %v", err)
	}

	_, err = jm.ValidateToken(t.Context(), token)
	if !errors.Is(err, ErrExpiredToken) {
		failf(t, "ValidateToken() with expired token: got %v, want ErrExpiredToken", err)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	jm1, _ := NewJWTManager("secret1", time.Hour)
	jm2, _ := NewJWTManager("secret2", time.Hour)

	token, err := jm1.CreateToken(t.Context(), "user-123")
	if err != nil {
		failNowf(t, "CreateToken() error: %v", err)
	}

	_, err = jm2.ValidateToken(t.Context(), token)
	if !errors.Is(err, ErrInvalidToken) {
		failf(t, "ValidateToken() with wrong secret: got %v, want ErrInvalidToken", err)
	}
}

func TestValidateToken_Empty(t *testing.T) {
	jm, _ := NewJWTManager("testsecret", time.Hour)
	_, err := jm.ValidateToken(t.Context(), "")
	if !errors.Is(err, ErrInvalidToken) {
		failf(t, "ValidateToken() with empty token: got %v, want ErrInvalidToken", err)
	}
}
