package cache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"odin/pkg/cache"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStrategyManager_GenerateKey(t *testing.T) {
	store := cache.NewMemoryStore()
	config := cache.CacheConfig{
		Strategy: cache.StrategyTTL,
		TTL:      5 * time.Minute,
	}

	sm := cache.NewStrategyManager(store, config, logrus.New())

	req := httptest.NewRequest("GET", "/api/users?page=1&limit=10", nil)
	key1 := sm.GenerateKey(req, "")

	req2 := httptest.NewRequest("GET", "/api/users?limit=10&page=1", nil)
	key2 := sm.GenerateKey(req2, "")

	assert.Equal(t, key1, key2, "Keys should be identical for equivalent requests")
}

func TestStrategyManager_ShouldCache(t *testing.T) {
	store := cache.NewMemoryStore()
	config := cache.CacheConfig{
		Strategy:        cache.StrategyTTL,
		CacheableStatus: []int{200, 404},
	}

	sm := cache.NewStrategyManager(store, config, logrus.New())

	headers := make(http.Header)
	assert.True(t, sm.ShouldCache(200, headers))
	assert.True(t, sm.ShouldCache(404, headers))
	assert.False(t, sm.ShouldCache(500, headers))

	headers.Set("Cache-Control", "no-cache")
	assert.False(t, sm.ShouldCache(200, headers))
}

func TestStrategyManager_SetAndGet(t *testing.T) {
	store := cache.NewMemoryStore()
	config := cache.CacheConfig{
		Strategy: cache.StrategyTTL,
		TTL:      5 * time.Minute,
	}

	sm := cache.NewStrategyManager(store, config, logrus.New())

	entry := &cache.CacheEntry{
		Data:       []byte(`{"test": true}`),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
	}

	key := "test-key"
	sm.Set(key, entry)

	retrieved, found := sm.Get(key)
	require.True(t, found)
	assert.Equal(t, entry.Data, retrieved.Data)
	assert.Equal(t, entry.StatusCode, retrieved.StatusCode)
	assert.NotZero(t, retrieved.Timestamp)
	assert.Equal(t, config.TTL, retrieved.TTL)
}

func TestStrategyManager_ConditionalCaching(t *testing.T) {
	store := cache.NewMemoryStore()
	config := cache.CacheConfig{
		Strategy: cache.StrategyConditional,
		TTL:      5 * time.Minute,
	}

	sm := cache.NewStrategyManager(store, config, logrus.New())

	entry := &cache.CacheEntry{
		Data:       []byte(`{"test": true}`),
		Headers:    map[string]string{"Cache-Control": "max-age=3600"},
		StatusCode: 200,
	}

	key := "conditional-test"
	sm.Set(key, entry)

	retrieved, found := sm.Get(key)
	require.True(t, found)
	assert.Equal(t, time.Hour, retrieved.TTL)
}

func TestStrategyManager_CheckConditional(t *testing.T) {
	store := cache.NewMemoryStore()
	config := cache.CacheConfig{Strategy: cache.StrategyConditional}

	sm := cache.NewStrategyManager(store, config, logrus.New())

	entry := &cache.CacheEntry{
		ETag:         `"abc123"`,
		LastModified: time.Now().Format(time.RFC1123),
	}

	tests := []struct {
		name           string
		headers        map[string]string
		expectModified bool
		expectStatus   int
	}{
		{
			name:           "no conditional headers",
			headers:        map[string]string{},
			expectModified: false,
			expectStatus:   0,
		},
		{
			name:           "matching etag",
			headers:        map[string]string{"If-None-Match": `"abc123"`},
			expectModified: true,
			expectStatus:   http.StatusNotModified,
		},
		{
			name:           "non-matching etag",
			headers:        map[string]string{"If-None-Match": `"def456"`},
			expectModified: false,
			expectStatus:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			modified, status := sm.CheckConditional(req, entry)
			assert.Equal(t, tt.expectModified, modified)
			assert.Equal(t, tt.expectStatus, status)
		})
	}
}
