package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewJWTManager_WithSecret(t *testing.T) {
	jm, err := NewJWTManager("mysecret", time.Hour)
	require.NoError(t, err, "NewJWTManager() unexpected error")
	require.NotNil(t, jm)
}

func TestNewJWTManager_RandomSecret(t *testing.T) {
	jm, err := NewJWTManager("", time.Hour)
	require.NoError(t, err, "NewJWTManager() with empty secret unexpected error")
	require.NotNil(t, jm)
	if len(jm.secret) != 32 {
		t.Errorf("expected 32-byte random secret, got %d bytes", len(jm.secret))
	}
}

func TestCreateAndValidateToken(t *testing.T) {
	jm, err := NewJWTManager("testsecret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")

	userID := "user-123"
	token, err := jm.CreateToken(t.Context(), userID)
	require.NoError(t, err, "CreateToken() error")
	require.NotEmpty(t, token)

	claims, err := jm.ValidateToken(t.Context(), token)
	require.NoError(t, err, "ValidateToken() error")
	if claims.UserID != userID {
		t.Errorf("ValidateToken() UserID = %q, want %q", claims.UserID, userID)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	jm, err := NewJWTManager("testsecret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")

	_, err = jm.ValidateToken(t.Context(), "not-a-valid-token")
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("ValidateToken() with invalid token: got %v, want ErrInvalidToken", err)
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Create manager with very short TTL
	jm, err := NewJWTManager("testsecret", -time.Second)
	require.NoError(t, err, "NewJWTManager() error")

	token, err := jm.CreateToken(t.Context(), "user-123")
	require.NoError(t, err, "CreateToken() error")

	_, err = jm.ValidateToken(t.Context(), token)
	if !errors.Is(err, ErrExpiredToken) {
		t.Errorf("ValidateToken() with expired token: got %v, want ErrExpiredToken", err)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	jm1, err := NewJWTManager("secret1", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")
	jm2, err := NewJWTManager("secret2", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")

	token, err := jm1.CreateToken(t.Context(), "user-123")
	require.NoError(t, err, "CreateToken() error")

	_, err = jm2.ValidateToken(t.Context(), token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("ValidateToken() with wrong secret: got %v, want ErrInvalidToken", err)
	}
}

func TestValidateToken_Empty(t *testing.T) {
	jm, err := NewJWTManager("testsecret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")
	_, err = jm.ValidateToken(t.Context(), "")
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("ValidateToken() with empty token: got %v, want ErrInvalidToken", err)
	}
}
