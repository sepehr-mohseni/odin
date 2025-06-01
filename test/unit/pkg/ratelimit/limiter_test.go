package ratelimit

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"odin/pkg/ratelimit"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLimiter(t *testing.T) {
	config := ratelimit.Config{
		Enabled:       true,
		Algorithm:     ratelimit.AlgorithmFixedWindow,
		DefaultLimit:  100,
		DefaultWindow: time.Minute,
	}

	limiter, err := ratelimit.NewLimiter(config, logrus.New())
	require.NoError(t, err)
	assert.NotNil(t, limiter)
}

func TestLimiter_ShouldSkip(t *testing.T) {
	config := ratelimit.Config{
		Enabled:   true,
		SkipPaths: []string{"/health", "/metrics"},
	}

	limiter, err := ratelimit.NewLimiter(config, logrus.New())
	require.NoError(t, err)

	e := echo.New()

	tests := []struct {
		path       string
		shouldSkip bool
	}{
		{"/health", true},
		{"/metrics", true},
		{"/health/check", true},
		{"/api/users", false},
		{"/healthz", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			middleware := limiter.Middleware()
			nextCalled := false

			next := func(c echo.Context) error {
				nextCalled = true
				return nil
			}

			err := middleware(next)(c)
			assert.NoError(t, err)
			assert.Equal(t, tt.shouldSkip, nextCalled)
		})
	}
}

func TestLimiter_GenerateKey(t *testing.T) {
	config := ratelimit.Config{
		HeadersToInclude: []string{"User-Agent"},
	}

	limiter, err := ratelimit.NewLimiter(config, logrus.New())
	require.NoError(t, err)

	e := echo.New()

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	rule := &ratelimit.Rule{
		Path:   "/api/test",
		Method: "GET",
	}

	key := limiter.GenerateKey(c, rule)
	assert.Contains(t, key, "api_key:test-key")
}

func TestLimiter_CheckLimit_FixedWindow(t *testing.T) {
	config := ratelimit.Config{
		Algorithm:     ratelimit.AlgorithmFixedWindow,
		DefaultLimit:  2,
		DefaultWindow: time.Minute,
	}

	limiter, err := ratelimit.NewLimiter(config, logrus.New())
	require.NoError(t, err)

	rule := &ratelimit.Rule{
		Limit:  2,
		Window: time.Minute,
	}

	key := "test-key"
	ctx := context.Background()

	limitInfo1, allowed1 := limiter.CheckLimit(ctx, key, rule)
	assert.True(t, allowed1)
	assert.Equal(t, 2, limitInfo1.Limit)
	assert.Equal(t, 1, limitInfo1.Remaining)

	limitInfo2, allowed2 := limiter.CheckLimit(ctx, key, rule)
	assert.True(t, allowed2)
	assert.Equal(t, 0, limitInfo2.Remaining)

	limitInfo3, allowed3 := limiter.CheckLimit(ctx, key, rule)
	assert.False(t, allowed3)
	assert.Equal(t, 0, limitInfo3.Remaining)
}

func TestLimiter_MatchesRule(t *testing.T) {
	config := ratelimit.Config{}

	limiter, err := ratelimit.NewLimiter(config, logrus.New())
	require.NoError(t, err)

	e := echo.New()

	tests := []struct {
		name        string
		rule        ratelimit.Rule
		method      string
		path        string
		headers     map[string]string
		expectMatch bool
	}{
		{
			name: "exact path match",
			rule: ratelimit.Rule{
				Method: "GET",
				Path:   "/api/users",
			},
			method:      "GET",
			path:        "/api/users",
			expectMatch: true,
		},
		{
			name: "prefix path match",
			rule: ratelimit.Rule{
				Method: "GET",
				Path:   "/api/",
			},
			method:      "GET",
			path:        "/api/users",
			expectMatch: true,
		},
		{
			name: "method mismatch",
			rule: ratelimit.Rule{
				Method: "POST",
				Path:   "/api/users",
			},
			method:      "GET",
			path:        "/api/users",
			expectMatch: false,
		},
		{
			name: "header rule match",
			rule: ratelimit.Rule{
				Path: "/api/admin",
				Headers: []ratelimit.HeaderRule{
					{Name: "X-Admin", Value: "true"},
				},
			},
			method:      "GET",
			path:        "/api/admin",
			headers:     map[string]string{"X-Admin": "true"},
			expectMatch: true,
		},
		{
			name: "header rule mismatch",
			rule: ratelimit.Rule{
				Path: "/api/admin",
				Headers: []ratelimit.HeaderRule{
					{Name: "X-Admin", Value: "true"},
				},
			},
			method:      "GET",
			path:        "/api/admin",
			headers:     map[string]string{"X-Admin": "false"},
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			matches := limiter.MatchesRule(c, &tt.rule)
			assert.Equal(t, tt.expectMatch, matches)
		})
	}
}

func TestLimiter_IPMatching(t *testing.T) {
	config := ratelimit.Config{}

	limiter, err := ratelimit.NewLimiter(config, logrus.New())
	require.NoError(t, err)

	tests := []struct {
		clientIP    string
		ruleIP      string
		expectMatch bool
	}{
		{"192.168.1.100", "192.168.1.100", true},
		{"192.168.1.100", "192.168.1.101", false},
		{"192.168.1.100", "192.168.1.0/24", true},
		{"10.0.0.1", "192.168.1.0/24", false},
		{"192.168.1.255", "192.168.1.0/24", true},
	}

	for _, tt := range tests {
		t.Run(tt.clientIP+"_vs_"+tt.ruleIP, func(t *testing.T) {
			matches := limiter.MatchesIP(tt.clientIP, tt.ruleIP)
			assert.Equal(t, tt.expectMatch, matches)
		})
	}
}
