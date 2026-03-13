package auth

import (
	"errors"
	"testing"
	"time"
)

func TestNewJWTManager_WithSecret(t *testing.T) {
	jm, err := NewJWTManager("mysecret", time.Hour)
	if err != nil {
		t.Fatalf("NewJWTManager() unexpected error: %v", err)
	}
	if jm == nil {
		t.Fatal("NewJWTManager() returned nil")
	}
}

func TestNewJWTManager_RandomSecret(t *testing.T) {
	jm, err := NewJWTManager("", time.Hour)
	if err != nil {
		t.Fatalf("NewJWTManager() with empty secret unexpected error: %v", err)
	}
	if jm == nil {
		t.Fatal("NewJWTManager() returned nil")
	}
	if len(jm.secret) != 32 {
		t.Errorf("expected 32-byte random secret, got %d bytes", len(jm.secret))
	}
}

func TestCreateAndValidateToken(t *testing.T) {
	jm, err := NewJWTManager("testsecret", time.Hour)
	if err != nil {
		t.Fatalf("NewJWTManager() error: %v", err)
	}

	userID := "user-123"
	token, err := jm.CreateToken(userID)
	if err != nil {
		t.Fatalf("CreateToken() error: %v", err)
	}
	if token == "" {
		t.Fatal("CreateToken() returned empty token")
	}

	claims, err := jm.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("ValidateToken() UserID = %q, want %q", claims.UserID, userID)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	jm, err := NewJWTManager("testsecret", time.Hour)
	if err != nil {
		t.Fatalf("NewJWTManager() error: %v", err)
	}

	_, err = jm.ValidateToken("not-a-valid-token")
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("ValidateToken() with invalid token: got %v, want ErrInvalidToken", err)
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Create manager with very short TTL
	jm, err := NewJWTManager("testsecret", -time.Second)
	if err != nil {
		t.Fatalf("NewJWTManager() error: %v", err)
	}

	token, err := jm.CreateToken("user-123")
	if err != nil {
		t.Fatalf("CreateToken() error: %v", err)
	}

	_, err = jm.ValidateToken(token)
	if !errors.Is(err, ErrExpiredToken) {
		t.Errorf("ValidateToken() with expired token: got %v, want ErrExpiredToken", err)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	jm1, _ := NewJWTManager("secret1", time.Hour)
	jm2, _ := NewJWTManager("secret2", time.Hour)

	token, err := jm1.CreateToken("user-123")
	if err != nil {
		t.Fatalf("CreateToken() error: %v", err)
	}

	_, err = jm2.ValidateToken(token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("ValidateToken() with wrong secret: got %v, want ErrInvalidToken", err)
	}
}

func TestValidateToken_Empty(t *testing.T) {
	jm, _ := NewJWTManager("testsecret", time.Hour)
	_, err := jm.ValidateToken("")
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("ValidateToken() with empty token: got %v, want ErrInvalidToken", err)
	}
}
