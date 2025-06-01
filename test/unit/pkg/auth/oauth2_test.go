package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"odin/pkg/auth"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
				Scopes:       []string{"openid", "email", "profile"},
				RedirectURL:  "http://localhost:8080/auth/callback",
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
				ClientID:    "test-client-id",
				AuthURL:     "https://accounts.google.com/o/oauth2/auth",
				Scopes:      []string{"openid", "email"},
				RedirectURL: "http://localhost:8080/auth/callback",
			},
		},
	}

	manager := auth.NewOAuth2Manager(config, logrus.New())

	tests := []struct {
		name     string
		provider string
		state    string
		wantErr  bool
	}{
		{
			name:     "valid provider",
			provider: "google",
			state:    "test-state",
			wantErr:  false,
		},
		{
			name:     "invalid provider",
			provider: "invalid",
			state:    "test-state",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := manager.GetAuthURL(tt.provider, tt.state)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, url)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, url)
				assert.Contains(t, url, "client_id=test-client-id")
				assert.Contains(t, url, "state=test-state")
			}
		})
	}
}

func TestOAuth2Manager_ExchangeCodeForToken(t *testing.T) {
	// Mock OAuth2 server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			tokenResponse := auth.OAuth2Token{
				AccessToken: "access-token-123",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(tokenResponse); err != nil {
				t.Errorf("Failed to encode JSON response: %v", err)
			}
		}
	}))
	defer server.Close()

	config := auth.OAuth2Config{
		Enabled: true,
		Providers: map[string]auth.OAuth2Provider{
			"test": {
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				TokenURL:     server.URL + "/token",
				RedirectURL:  "http://localhost:8080/callback",
			},
		},
	}

	manager := auth.NewOAuth2Manager(config, logrus.New())

	token, err := manager.ExchangeCodeForToken(context.Background(), "test", "test-code")

	require.NoError(t, err)
	assert.Equal(t, "access-token-123", token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)
	assert.True(t, time.Now().Before(token.ExpiresAt))
}

func TestOAuth2Manager_ValidateToken(t *testing.T) {
	config := auth.OAuth2Config{
		Enabled: true,
	}

	manager := auth.NewOAuth2Manager(config, logrus.New())

	tests := []struct {
		name     string
		token    *auth.OAuth2Token
		expected bool
	}{
		{
			name: "valid token",
			token: &auth.OAuth2Token{
				AccessToken: "valid-token",
				ExpiresAt:   time.Now().Add(time.Hour),
			},
			expected: true,
		},
		{
			name: "expired token",
			token: &auth.OAuth2Token{
				AccessToken: "expired-token",
				ExpiresAt:   time.Now().Add(-time.Hour),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.ValidateToken(context.Background(), "test", tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}
