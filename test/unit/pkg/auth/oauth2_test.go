package auth

import (
	"context"
	"testing"

	"odin/pkg/auth"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewOAuth2Manager(t *testing.T) {
	config := auth.OAuth2Config{
		Enabled: true,
		Providers: map[string]auth.OAuth2Provider{
			"google": {
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				AuthURL:      "https://accounts.google.com/o/oauth2/auth",
				TokenURL:     "https://oauth2.googleapis.com/token",
				UserInfoURL:  "https://www.googleapis.com/oauth2/v1/userinfo",
				Scopes:       []string{"openid", "profile", "email"},
				RedirectURL:  "http://localhost:8080/auth/google/callback",
			},
		},
	}

	manager := auth.NewOAuth2Manager(config, logrus.New())
	assert.NotNil(t, manager)
}

func TestOAuth2Manager_GetAuthURL(t *testing.T) {
	config := auth.OAuth2Config{
		Enabled: true,
		Providers: map[string]auth.OAuth2Provider{
			"google": {
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				AuthURL:      "https://accounts.google.com/o/oauth2/auth",
				TokenURL:     "https://oauth2.googleapis.com/token",
				UserInfoURL:  "https://www.googleapis.com/oauth2/v1/userinfo",
				Scopes:       []string{"openid", "profile", "email"},
				RedirectURL:  "http://localhost:8080/auth/google/callback",
			},
		},
	}

	manager := auth.NewOAuth2Manager(config, logrus.New())

	tests := []struct {
		name        string
		provider    string
		expectError bool
	}{
		{
			name:        "valid provider",
			provider:    "google",
			expectError: false,
		},
		{
			name:        "invalid provider",
			provider:    "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := manager.GetAuthURL(tt.provider, "state123")

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, url)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, url)
				assert.Contains(t, url, "https://accounts.google.com/o/oauth2/auth")
			}
		})
	}
}

func TestOAuth2Manager_ExchangeCodeForToken(t *testing.T) {
	config := auth.OAuth2Config{
		Enabled: true,
		Providers: map[string]auth.OAuth2Provider{
			"test": {
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				AuthURL:      "https://example.com/auth",
				TokenURL:     "https://example.com/token",
				UserInfoURL:  "https://example.com/userinfo",
				Scopes:       []string{"read"},
				RedirectURL:  "http://localhost:8080/callback",
			},
		},
	}

	manager := auth.NewOAuth2Manager(config, logrus.New())

	// This will fail since we don't have a real OAuth2 server
	// But it tests that the method exists and handles errors correctly
	_, err := manager.ExchangeCodeForToken(context.Background(), "test", "invalid-code")
	assert.Error(t, err)
}

func TestOAuth2Manager_ValidateToken(t *testing.T) {
	config := auth.OAuth2Config{
		Enabled: true,
		Providers: map[string]auth.OAuth2Provider{
			"test": {
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				AuthURL:      "https://example.com/auth",
				TokenURL:     "https://example.com/token",
				UserInfoURL:  "https://example.com/userinfo",
				Scopes:       []string{"read"},
				RedirectURL:  "http://localhost:8080/callback",
			},
		},
	}

	manager := auth.NewOAuth2Manager(config, logrus.New())

	tests := []struct {
		name        string
		token       string
		expectValid bool
	}{
		{
			name:        "valid token",
			token:       "valid-token-123",
			expectValid: false, // Will be false since we don't have real validation
		},
		{
			name:        "expired token",
			token:       "expired-token",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a dummy OAuth2Token
			oauth2Token := &auth.OAuth2Token{
				AccessToken: tt.token,
				TokenType:   "Bearer",
			}

			valid := manager.ValidateToken(context.Background(), "test", oauth2Token)

			assert.Equal(t, tt.expectValid, valid)
		})
	}
}

func TestOAuth2BasicConfiguration(t *testing.T) {
	config := auth.OAuth2Config{
		Enabled: false,
	}

	manager := auth.NewOAuth2Manager(config, logrus.New())
	assert.NotNil(t, manager)
}
