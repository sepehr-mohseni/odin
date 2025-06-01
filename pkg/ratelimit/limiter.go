package ratelimit

import (
	"context"
	"crypto/md5"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type Algorithm string

const (
	AlgorithmTokenBucket   Algorithm = "token_bucket"
	AlgorithmSlidingWindow Algorithm = "sliding_window"
	AlgorithmFixedWindow   Algorithm = "fixed_window"
	AlgorithmLeakyBucket   Algorithm = "leaky_bucket"
)

type Config struct {
	Enabled          bool          `yaml:"enabled"`
	Algorithm        Algorithm     `yaml:"algorithm"`
	DefaultLimit     int           `yaml:"defaultLimit"`
	DefaultWindow    time.Duration `yaml:"defaultWindow"`
	BurstSize        int           `yaml:"burstSize"`
	Redis            RedisConfig   `yaml:"redis"`
	Rules            []Rule        `yaml:"rules"`
	SkipPaths        []string      `yaml:"skipPaths"`
	TrustedProxies   []string      `yaml:"trustedProxies"`
	HeadersToInclude []string      `yaml:"headersToInclude"`
	CustomKeyFunc    string        `yaml:"customKeyFunc"`
	ResponseHeaders  bool          `yaml:"responseHeaders"`
	LogViolations    bool          `yaml:"logViolations"`
}

type RedisConfig struct {
	Address  string `yaml:"address"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type Rule struct {
	Name        string        `yaml:"name"`
	Path        string        `yaml:"path"`
	Method      string        `yaml:"method"`
	Limit       int           `yaml:"limit"`
	Window      time.Duration `yaml:"window"`
	BurstSize   int           `yaml:"burstSize"`
	UserTypes   []string      `yaml:"userTypes"`
	APIKeys     []string      `yaml:"apiKeys"`
	IPWhitelist []string      `yaml:"ipWhitelist"`
	IPBlacklist []string      `yaml:"ipBlacklist"`
	Headers     []HeaderRule  `yaml:"headers"`
	SkipAuth    bool          `yaml:"skipAuth"`
}

type HeaderRule struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type KeyType string

const (
	KeyTypeIP     KeyType = "ip"
	KeyTypeUser   KeyType = "user"
	KeyTypeAPIKey KeyType = "api_key"
	KeyTypeCustom KeyType = "custom"
)

type Limiter struct {
	config      Config
	redisClient *redis.Client
	logger      *logrus.Logger
	rules       map[string]Rule
}

type LimitInfo struct {
	Key       string        `json:"key"`
	Limit     int           `json:"limit"`
	Remaining int           `json:"remaining"`
	ResetTime time.Time     `json:"reset_time"`
	Window    time.Duration `json:"window"`
}

type RateLimiter interface {
	Allow(key string) bool
	CheckLimit(ctx context.Context, key string, rule *Rule) (*LimitInfo, bool)
	Middleware() echo.MiddlewareFunc
}

func NewLimiter(config Config, logger *logrus.Logger) (*Limiter, error) {
	var redisClient *redis.Client

	if config.Redis.Address != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     config.Redis.Address,
			Password: config.Redis.Password,
			DB:       config.Redis.DB,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := redisClient.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to Redis: %w", err)
		}
	}

	rules := make(map[string]Rule)
	for _, rule := range config.Rules {
		key := fmt.Sprintf("%s:%s", rule.Method, rule.Path)
		rules[key] = rule
	}

	return &Limiter{
		config:      config,
		redisClient: redisClient,
		logger:      logger,
		rules:       rules,
	}, nil
}

func (l *Limiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !l.config.Enabled {
				return next(c)
			}

			if l.shouldSkip(c) {
				return next(c)
			}

			rule := l.findRule(c)
			if rule == nil {
				rule = &Rule{
					Limit:  l.config.DefaultLimit,
					Window: l.config.DefaultWindow,
				}
			}

			if l.isWhitelisted(c, rule) {
				return next(c)
			}

			if l.isBlacklisted(c, rule) {
				return echo.NewHTTPError(http.StatusForbidden, "IP address is blacklisted")
			}

			key := l.generateKey(c, rule)
			limitInfo, allowed := l.checkLimit(c.Request().Context(), key, rule)

			if l.config.ResponseHeaders {
				l.setHeaders(c, limitInfo)
			}

			if !allowed {
				if l.config.LogViolations {
					l.logger.WithFields(logrus.Fields{
						"key":       key,
						"limit":     limitInfo.Limit,
						"remaining": limitInfo.Remaining,
						"ip":        l.getClientIP(c),
						"path":      c.Request().URL.Path,
						"method":    c.Request().Method,
					}).Warn("Rate limit exceeded")
				}

				return echo.NewHTTPError(http.StatusTooManyRequests, "Rate limit exceeded")
			}

			return next(c)
		}
	}
}

func (l *Limiter) shouldSkip(c echo.Context) bool {
	path := c.Request().URL.Path

	for _, skipPath := range l.config.SkipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}

	return false
}

func (l *Limiter) findRule(c echo.Context) *Rule {
	method := c.Request().Method
	path := c.Request().URL.Path

	key := fmt.Sprintf("%s:%s", method, path)
	if rule, exists := l.rules[key]; exists {
		return &rule
	}

	for _, rule := range l.config.Rules {
		if l.MatchesRule(c, &rule) {
			return &rule
		}
	}

	return nil
}

func (l *Limiter) MatchesRule(c echo.Context, rule *Rule) bool {
	if rule.Method != "" && rule.Method != c.Request().Method {
		return false
	}

	if rule.Path != "" && !strings.HasPrefix(c.Request().URL.Path, rule.Path) {
		return false
	}

	for _, headerRule := range rule.Headers {
		headerValue := c.Request().Header.Get(headerRule.Name)
		if headerValue != headerRule.Value {
			return false
		}
	}

	return true
}

func (l *Limiter) isWhitelisted(c echo.Context, rule *Rule) bool {
	clientIP := l.getClientIP(c)

	for _, whiteIP := range rule.IPWhitelist {
		if l.MatchesIP(clientIP, whiteIP) {
			return true
		}
	}

	return false
}

func (l *Limiter) isBlacklisted(c echo.Context, rule *Rule) bool {
	clientIP := l.getClientIP(c)

	for _, blackIP := range rule.IPBlacklist {
		if l.MatchesIP(clientIP, blackIP) {
			return true
		}
	}

	return false
}

func (l *Limiter) MatchesIP(clientIP, ruleIP string) bool {
	if strings.Contains(ruleIP, "/") {
		_, network, err := net.ParseCIDR(ruleIP)
		if err != nil {
			return false
		}
		ip := net.ParseIP(clientIP)
		return network.Contains(ip)
	}

	return clientIP == ruleIP
}

func (l *Limiter) generateKey(c echo.Context, rule *Rule) string {
	var keyParts []string

	if apiKey := c.Request().Header.Get("X-API-Key"); apiKey != "" {
		keyParts = append(keyParts, "api_key:"+apiKey)
	} else if userID := l.getUserID(c); userID != "" {
		keyParts = append(keyParts, "user:"+userID)
	} else {
		keyParts = append(keyParts, "ip:"+l.getClientIP(c))
	}

	keyParts = append(keyParts, rule.Path)
	keyParts = append(keyParts, rule.Method)

	for _, header := range l.config.HeadersToInclude {
		value := c.Request().Header.Get(header)
		if value != "" {
			keyParts = append(keyParts, fmt.Sprintf("%s:%s", header, value))
		}
	}

	key := strings.Join(keyParts, ":")
	hash := md5.Sum([]byte(key))
	return fmt.Sprintf("ratelimit:%x", hash)
}

func (l *Limiter) getClientIP(c echo.Context) string {
	if l.isTrustedProxy(c.RealIP()) {
		if xff := c.Request().Header.Get("X-Forwarded-For"); xff != "" {
			ips := strings.Split(xff, ",")
			return strings.TrimSpace(ips[0])
		}
		if xri := c.Request().Header.Get("X-Real-IP"); xri != "" {
			return xri
		}
	}

	return c.RealIP()
}

func (l *Limiter) isTrustedProxy(ip string) bool {
	for _, trustedIP := range l.config.TrustedProxies {
		if l.MatchesIP(ip, trustedIP) {
			return true
		}
	}
	return false
}

func (l *Limiter) getUserID(c echo.Context) string {
	if user := c.Get("user"); user != nil {
		if userMap, ok := user.(map[string]interface{}); ok {
			if userID, exists := userMap["user_id"]; exists {
				return fmt.Sprintf("%v", userID)
			}
		}
	}

	return ""
}

func (l *Limiter) CheckLimit(ctx context.Context, key string, rule *Rule) (*LimitInfo, bool) {
	limit := rule.Limit
	window := rule.Window

	if limit <= 0 {
		limit = l.config.DefaultLimit
	}
	if window <= 0 {
		window = l.config.DefaultWindow
	}

	switch l.config.Algorithm {
	case AlgorithmSlidingWindow:
		return l.checkSlidingWindow(ctx, key, limit, window)
	case AlgorithmFixedWindow:
		return l.checkFixedWindow(ctx, key, limit, window)
	case AlgorithmTokenBucket:
		return l.checkTokenBucket(ctx, key, limit, window, rule.BurstSize)
	default:
		return l.checkFixedWindow(ctx, key, limit, window)
	}
}

func (l *Limiter) checkFixedWindow(ctx context.Context, key string, limit int, window time.Duration) (*LimitInfo, bool) {
	if l.redisClient == nil {
		return &LimitInfo{
			Key:       key,
			Limit:     limit,
			Remaining: limit - 1,
			ResetTime: time.Now().Add(window),
			Window:    window,
		}, true
	}

	windowKey := fmt.Sprintf("%s:%d", key, time.Now().Unix()/int64(window.Seconds()))

	pipe := l.redisClient.Pipeline()
	incrCmd := pipe.Incr(ctx, windowKey)
	pipe.Expire(ctx, windowKey, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		l.logger.WithError(err).Error("Redis pipeline execution failed")
		return &LimitInfo{Key: key, Limit: limit, Remaining: 0}, false
	}

	count := int(incrCmd.Val())
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	resetTime := time.Now().Add(window)

	return &LimitInfo{
		Key:       key,
		Limit:     limit,
		Remaining: remaining,
		ResetTime: resetTime,
		Window:    window,
	}, count <= limit
}

func (l *Limiter) checkSlidingWindow(ctx context.Context, key string, limit int, window time.Duration) (*LimitInfo, bool) {
	if l.redisClient == nil {
		return &LimitInfo{
			Key:       key,
			Limit:     limit,
			Remaining: limit - 1,
			ResetTime: time.Now().Add(window),
			Window:    window,
		}, true
	}

	now := time.Now()
	windowStart := now.Add(-window)

	pipe := l.redisClient.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart.UnixNano(), 10))
	countCmd := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixNano()), Member: now.UnixNano()})
	pipe.Expire(ctx, key, window+time.Minute)

	_, err := pipe.Exec(ctx)
	if err != nil {
		l.logger.WithError(err).Error("Redis pipeline execution failed")
		return &LimitInfo{Key: key, Limit: limit, Remaining: 0}, false
	}

	count := int(countCmd.Val())
	remaining := limit - count - 1
	if remaining < 0 {
		remaining = 0
	}

	return &LimitInfo{
		Key:       key,
		Limit:     limit,
		Remaining: remaining,
		ResetTime: now.Add(window),
		Window:    window,
	}, count < limit
}

func (l *Limiter) checkTokenBucket(ctx context.Context, key string, limit int, window time.Duration, burstSize int) (*LimitInfo, bool) {
	if burstSize <= 0 {
		burstSize = l.config.BurstSize
	}
	if burstSize <= 0 {
		burstSize = limit
	}

	return l.checkFixedWindow(ctx, key, burstSize, window)
}

func (l *Limiter) setHeaders(c echo.Context, limitInfo *LimitInfo) {
	c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(limitInfo.Limit))
	c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(limitInfo.Remaining))
	c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(limitInfo.ResetTime.Unix(), 10))
	c.Response().Header().Set("X-RateLimit-Window", limitInfo.Window.String())
}

func (l *Limiter) Close() error {
	if l.redisClient != nil {
		return l.redisClient.Close()
	}
	return nil
}

func (l *Limiter) checkLimit(ctx context.Context, key string, rule *Rule) (*LimitInfo, bool) {
	return l.CheckLimit(ctx, key, rule)
}

func (l *Limiter) GenerateKey(c echo.Context, rule *Rule) string {
	return l.generateKey(c, rule)
}

func (l *Limiter) Allow(key string) bool {
	rule := &Rule{
		Limit:  l.config.DefaultLimit,
		Window: l.config.DefaultWindow,
	}

	_, allowed := l.checkLimit(context.Background(), key, rule)
	return allowed
}
