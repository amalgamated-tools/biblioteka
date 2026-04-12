package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestNewJWTManager_WithSecret(t *testing.T) {
	jm, err := NewJWTManager("mysecret", time.Hour)
	require.NoError(t, err, "NewJWTManager() unexpected error")
	require.NotNil(t, jm)
}

func TestMinSecretLength(t *testing.T) {
	require.Equal(t, 32, MinSecretLength, "MinSecretLength should be 32")
}

func TestNewJWTManager_ShortSecret(t *testing.T) {
	// A short (non-empty) secret is accepted — the caller is responsible for
	// logging a warning — but the manager must still function correctly.
	jm, err := NewJWTManager("short", time.Hour)
	require.NoError(t, err, "NewJWTManager() with short secret should succeed")
	require.NotNil(t, jm)

	token, err := jm.CreateToken(t.Context(), "user-1")
	require.NoError(t, err, "CreateToken() with short secret")
	claims, err := jm.ValidateToken(t.Context(), token)
	require.NoError(t, err, "ValidateToken() with short secret")
	require.Equal(t, "user-1", claims.UserID)
}

func TestNewJWTManager_RandomSecret(t *testing.T) {
	jm, err := NewJWTManager("", time.Hour)
	require.NoError(t, err, "NewJWTManager() with empty secret unexpected error")
	require.NotNil(t, jm)
	require.Len(t, jm.secret, 32)
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
	require.Equal(t, userID, claims.UserID)
	require.Equal(t, jwtIssuer, claims.Issuer)
	require.Equal(t, jwt.ClaimStrings{jwtIssuer}, claims.Audience)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	jm, err := NewJWTManager("testsecret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")

	_, err = jm.ValidateToken(t.Context(), "not-a-valid-token")
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Create manager with very short TTL
	jm, err := NewJWTManager("testsecret", -time.Second)
	require.NoError(t, err, "NewJWTManager() error")

	token, err := jm.CreateToken(t.Context(), "user-123")
	require.NoError(t, err, "CreateToken() error")

	_, err = jm.ValidateToken(t.Context(), token)
	require.ErrorIs(t, err, ErrExpiredToken)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	jm1, err := NewJWTManager("secret1", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")
	jm2, err := NewJWTManager("secret2", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")

	token, err := jm1.CreateToken(t.Context(), "user-123")
	require.NoError(t, err, "CreateToken() error")

	_, err = jm2.ValidateToken(t.Context(), token)
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_Empty(t *testing.T) {
	jm, err := NewJWTManager("testsecret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")
	_, err = jm.ValidateToken(t.Context(), "")
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_WrongIssuer(t *testing.T) {
	jm, err := NewJWTManager("testsecret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")

	// Craft a token with a different issuer using the same secret.
	now := time.Now()
	claims := Claims{
		UserID: "user-123",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "other-service",
			Audience:  jwt.ClaimStrings{jwtIssuer},
			Subject:   "user-123",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(jm.secret)
	require.NoError(t, err, "sign token with wrong issuer")

	_, err = jm.ValidateToken(t.Context(), signed)
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_WrongAudience(t *testing.T) {
	jm, err := NewJWTManager("testsecret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")

	// Craft a token with a different audience using the same secret.
	now := time.Now()
	claims := Claims{
		UserID: "user-123",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Audience:  jwt.ClaimStrings{"other-service"},
			Subject:   "user-123",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(jm.secret)
	require.NoError(t, err, "sign token with wrong audience")

	_, err = jm.ValidateToken(t.Context(), signed)
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_MissingIssuerAndAudience(t *testing.T) {
	jm, err := NewJWTManager("testsecret", time.Hour)
	require.NoError(t, err)

	// Simulate a token created before iss/aud were introduced.
	now := time.Now()
	claims := Claims{
		UserID: "user-123",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-123",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			// Issuer and Audience intentionally absent
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(jm.secret)
	require.NoError(t, err)

	_, err = jm.ValidateToken(t.Context(), signed)
	require.ErrorIs(t, err, ErrInvalidToken, "legacy token without iss/aud must be rejected")
}
