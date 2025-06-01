package cache

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type CacheStrategy string

const (
	StrategyTTL          CacheStrategy = "ttl"
	StrategyConditional  CacheStrategy = "conditional"
	StrategyVary         CacheStrategy = "vary"
	StrategyUserContext  CacheStrategy = "user_context"
	StrategyInvalidation CacheStrategy = "invalidation"
)

type CacheConfig struct {
	Strategy         CacheStrategy `yaml:"strategy"`
	TTL              time.Duration `yaml:"ttl"`
	VaryHeaders      []string      `yaml:"varyHeaders"`
	CacheableStatus  []int         `yaml:"cacheableStatus"`
	IgnoreHeaders    []string      `yaml:"ignoreHeaders"`
	UserContextKey   string        `yaml:"userContextKey"`
	InvalidationTags []string      `yaml:"invalidationTags"`
	MaxSize          int64         `yaml:"maxSize"`
	CompressResponse bool          `yaml:"compressResponse"`
}

type CacheEntry struct {
	Data         []byte            `json:"data"`
	Headers      map[string]string `json:"headers"`
	StatusCode   int               `json:"status_code"`
	Timestamp    time.Time         `json:"timestamp"`
	TTL          time.Duration     `json:"ttl"`
	ETag         string            `json:"etag"`
	LastModified string            `json:"last_modified"`
	Tags         []string          `json:"tags"`
	UserContext  string            `json:"user_context"`
}

type StrategyManager struct {
	store  Store
	config CacheConfig
	logger *logrus.Logger
}

func NewStrategyManager(store Store, config CacheConfig, logger *logrus.Logger) *StrategyManager {
	if len(config.CacheableStatus) == 0 {
		config.CacheableStatus = []int{200, 301, 302, 404}
	}

	return &StrategyManager{
		store:  store,
		config: config,
		logger: logger,
	}
}

func (sm *StrategyManager) GenerateKey(req *http.Request, userContext string) string {
	var keyParts []string

	keyParts = append(keyParts, req.Method)
	keyParts = append(keyParts, req.URL.Path)

	if req.URL.RawQuery != "" {
		params, err := url.ParseQuery(req.URL.RawQuery)
		if err == nil {
			var sortedParams []string
			for key, values := range params {
				for _, value := range values {
					sortedParams = append(sortedParams, fmt.Sprintf("%s=%s", key, value))
				}
			}
			sort.Strings(sortedParams)
			keyParts = append(keyParts, strings.Join(sortedParams, "&"))
		}
	}

	if sm.config.Strategy == StrategyVary && len(sm.config.VaryHeaders) > 0 {
		var varyParts []string
		for _, header := range sm.config.VaryHeaders {
			value := req.Header.Get(header)
			if value != "" {
				varyParts = append(varyParts, fmt.Sprintf("%s:%s", header, value))
			}
		}
		if len(varyParts) > 0 {
			keyParts = append(keyParts, strings.Join(varyParts, "|"))
		}
	}

	if sm.config.Strategy == StrategyUserContext && userContext != "" {
		keyParts = append(keyParts, "user:"+userContext)
	}

	key := strings.Join(keyParts, ":")
	hash := md5.Sum([]byte(key))
	return fmt.Sprintf("cache:%x", hash)
}

func (sm *StrategyManager) ShouldCache(statusCode int, headers http.Header) bool {
	// Check cache control headers first
	if cacheControl := headers.Get("Cache-Control"); cacheControl != "" {
		if strings.Contains(cacheControl, "no-cache") || strings.Contains(cacheControl, "no-store") {
			return false
		}
	}

	// Check if status code is cacheable
	for _, status := range sm.config.CacheableStatus {
		if status == statusCode {
			return true
		}
	}

	return false
}

func (sm *StrategyManager) Get(key string) (*CacheEntry, bool) {
	data, found := sm.store.Get(key)
	if !found {
		return nil, false
	}

	entry, ok := data.(*CacheEntry)
	if !ok {
		sm.logger.WithField("key", key).Warn("Invalid cache entry type")
		sm.store.Delete(key)
		return nil, false
	}

	if sm.isExpired(entry) {
		sm.store.Delete(key)
		return nil, false
	}

	return entry, true
}

func (sm *StrategyManager) Set(key string, entry *CacheEntry) {
	var ttl time.Duration

	switch sm.config.Strategy {
	case StrategyTTL:
		ttl = sm.config.TTL
	case StrategyConditional:
		ttl = sm.getConditionalTTL(entry)
	default:
		ttl = sm.config.TTL
	}

	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	entry.TTL = ttl
	entry.Timestamp = time.Now()

	sm.store.Set(key, entry, ttl)

	sm.logger.WithFields(logrus.Fields{
		"key":      key,
		"ttl":      ttl,
		"strategy": sm.config.Strategy,
	}).Debug("Cache entry stored")
}

func (sm *StrategyManager) isExpired(entry *CacheEntry) bool {
	return time.Since(entry.Timestamp) > entry.TTL
}

func (sm *StrategyManager) getConditionalTTL(entry *CacheEntry) time.Duration {
	if entry.Headers["Cache-Control"] != "" {
		cacheControl := entry.Headers["Cache-Control"]

		if strings.Contains(cacheControl, "max-age=") {
			parts := strings.Split(cacheControl, "max-age=")
			if len(parts) > 1 {
				maxAgeStr := strings.Split(parts[1], ",")[0]
				maxAgeStr = strings.TrimSpace(maxAgeStr)
				if seconds := parseInt(maxAgeStr); seconds > 0 {
					return time.Duration(seconds) * time.Second
				}
			}
		}
	}

	if entry.Headers["Expires"] != "" {
		if expires, err := time.Parse(time.RFC1123, entry.Headers["Expires"]); err == nil {
			ttl := time.Until(expires)
			if ttl > 0 {
				return ttl
			}
		}
	}

	return sm.config.TTL
}

func (sm *StrategyManager) InvalidateByTags(tags []string) {
	sm.logger.WithField("tags", tags).Info("Invalidating cache by tags")
	sm.store.Clear()
}

func (sm *StrategyManager) InvalidateByPattern(pattern string) {
	sm.logger.WithField("pattern", pattern).Info("Invalidating cache by pattern")
	sm.store.Clear()
}

func (sm *StrategyManager) CheckConditional(req *http.Request, entry *CacheEntry) (bool, int) {
	ifNoneMatch := req.Header.Get("If-None-Match")
	if ifNoneMatch != "" && entry.ETag != "" {
		if ifNoneMatch == entry.ETag || ifNoneMatch == "*" {
			return true, http.StatusNotModified
		}
	}

	ifModifiedSince := req.Header.Get("If-Modified-Since")
	if ifModifiedSince != "" && entry.LastModified != "" {
		if clientTime, err := time.Parse(time.RFC1123, ifModifiedSince); err == nil {
			if serverTime, err := time.Parse(time.RFC1123, entry.LastModified); err == nil {
				if !serverTime.After(clientTime) {
					return true, http.StatusNotModified
				}
			}
		}
	}

	return false, 0
}

func parseInt(s string) int {
	var result int
	for _, char := range s {
		if char >= '0' && char <= '9' {
			result = result*10 + int(char-'0')
		} else {
			break
		}
	}
	return result
}

type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]interface{}
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		items: make(map[string]interface{}),
	}
}

func (m *MemoryStore) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, exists := m.items[key]
	return value, exists
}

func (m *MemoryStore) Set(key string, value interface{}, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[key] = value
}

func (m *MemoryStore) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, key)
}

func (m *MemoryStore) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[string]interface{})
}

func (m *MemoryStore) Close() error {
	return nil
}
