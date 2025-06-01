package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type OAuth2Config struct {
	Enabled      bool                      `yaml:"enabled"`
	Providers    map[string]OAuth2Provider `yaml:"providers"`
	DefaultScope string                    `yaml:"defaultScope"`
	TokenTTL     time.Duration             `yaml:"tokenTTL"`
}

type OAuth2Provider struct {
	ClientID     string   `yaml:"clientId"`
	ClientSecret string   `yaml:"clientSecret"`
	AuthURL      string   `yaml:"authUrl"`
	TokenURL     string   `yaml:"tokenUrl"`
	UserInfoURL  string   `yaml:"userInfoUrl"`
	Scopes       []string `yaml:"scopes"`
	RedirectURL  string   `yaml:"redirectUrl"`
}

type OAuth2Manager struct {
	config    OAuth2Config
	logger    *logrus.Logger
	providers map[string]*OAuth2Provider
}

type OAuth2Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type OAuth2UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

func NewOAuth2Manager(config OAuth2Config, logger *logrus.Logger) *OAuth2Manager {
	providers := make(map[string]*OAuth2Provider)
	for name, provider := range config.Providers {
		providerCopy := provider
		providers[name] = &providerCopy
	}

	return &OAuth2Manager{
		config:    config,
		logger:    logger,
		providers: providers,
	}
}

func (om *OAuth2Manager) GetAuthURL(provider, state string) (string, error) {
	p, exists := om.providers[provider]
	if !exists {
		return "", fmt.Errorf("provider %s not found", provider)
	}

	params := url.Values{
		"client_id":     {p.ClientID},
		"redirect_uri":  {p.RedirectURL},
		"response_type": {"code"},
		"scope":         {strings.Join(p.Scopes, " ")},
		"state":         {state},
	}

	return p.AuthURL + "?" + params.Encode(), nil
}

func (om *OAuth2Manager) ExchangeCodeForToken(ctx context.Context, provider, code string) (*OAuth2Token, error) {
	p, exists := om.providers[provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", provider)
	}

	data := url.Values{
		"client_id":     {p.ClientID},
		"client_secret": {p.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {p.RedirectURL},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status: %d", resp.StatusCode)
	}

	var token OAuth2Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return &token, nil
}

func (om *OAuth2Manager) GetUserInfo(ctx context.Context, provider string, token *OAuth2Token) (*OAuth2UserInfo, error) {
	p, exists := om.providers[provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", provider)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", p.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed with status: %d", resp.StatusCode)
	}

	var userInfo OAuth2UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	userInfo.Provider = provider
	return &userInfo, nil
}

func (om *OAuth2Manager) ValidateToken(ctx context.Context, provider string, token *OAuth2Token) bool {
	if time.Now().After(token.ExpiresAt) {
		return false
	}

	// Additional validation can be added here
	return true
}

func OAuth2Middleware(manager *OAuth2Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization header format")
			}

			// Here you would validate the OAuth2 token
			// This is a simplified version - in practice, you'd need to
			// validate against the provider or store token information

			return next(c)
		}
	}
}
