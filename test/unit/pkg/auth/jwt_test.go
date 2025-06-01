package auth

import (
	"testing"
	"time"

	"odin/pkg/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWTManager(t *testing.T) {
	config := auth.JWTConfig{
		Secret:          "test-secret",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
	}

	manager := auth.NewJWTManager(config)
	assert.NotNil(t, manager)
}

func TestGenerateToken(t *testing.T) {
	config := auth.JWTConfig{
		Secret:          "test-secret",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
	}

	manager := auth.NewJWTManager(config)

	claims := &auth.Claims{
		UserID:   "user123",
		Username: "testuser",
		Role:     "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token, err := manager.GenerateToken(claims)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateToken(t *testing.T) {
	config := auth.JWTConfig{
		Secret:          "test-secret",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
	}

	manager := auth.NewJWTManager(config)

	// Generate a token
	claims := &auth.Claims{
		UserID:   "user123",
		Username: "testuser",
		Role:     "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	tokenString, err := manager.GenerateToken(claims)
	require.NoError(t, err)

	// Validate the token
	validatedClaims, err := manager.ValidateToken(tokenString)
	require.NoError(t, err)
	assert.Equal(t, claims.UserID, validatedClaims.UserID)
	assert.Equal(t, claims.Username, validatedClaims.Username)
	assert.Equal(t, claims.Role, validatedClaims.Role)
}

func TestValidateExpiredToken(t *testing.T) {
	config := auth.JWTConfig{
		Secret:          "test-secret",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
	}

	manager := auth.NewJWTManager(config)

	// Generate an expired token
	claims := &auth.Claims{
		UserID:   "user123",
		Username: "testuser",
		Role:     "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // Expired
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	tokenString, err := manager.GenerateToken(claims)
	require.NoError(t, err)

	// Validate the expired token
	_, err = manager.ValidateToken(tokenString)
	assert.Error(t, err)
}

func TestValidateInvalidToken(t *testing.T) {
	config := auth.JWTConfig{
		Secret:          "test-secret",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
	}

	manager := auth.NewJWTManager(config)

	// Validate an invalid token
	_, err := manager.ValidateToken("invalid.token.here")
	assert.Error(t, err)
}
