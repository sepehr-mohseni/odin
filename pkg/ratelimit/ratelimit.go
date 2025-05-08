package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	Allow(key string) bool
}

// Config holds rate limiter configuration
type Config struct {
	Enabled  bool
	Limit    int
	Duration time.Duration
	Strategy string
	RedisURL string
}

// LocalLimiter implements a simple in-memory rate limiter
type LocalLimiter struct {
	limit     int
	duration  time.Duration
	requests  map[string][]time.Time
	mutex     sync.Mutex
	logger    *logrus.Logger
	cleanupAt time.Time
}

// NewLocalLimiter creates a new in-memory rate limiter
func NewLocalLimiter(limit int, duration time.Duration, logger *logrus.Logger) *LocalLimiter {
	return &LocalLimiter{
		limit:     limit,
		duration:  duration,
		requests:  make(map[string][]time.Time),
		mutex:     sync.Mutex{},
		logger:    logger,
		cleanupAt: time.Now().Add(duration),
	}
}

// Allow checks if a request is allowed based on rate limits
func (l *LocalLimiter) Allow(key string) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	now := time.Now()

	// Cleanup expired entries periodically
	if now.After(l.cleanupAt) {
		l.cleanup(now)
		l.cleanupAt = now.Add(l.duration)
	}

	// Get existing timestamps for this key
	timestamps, exists := l.requests[key]
	if !exists {
		timestamps = []time.Time{}
	}

	// Filter out timestamps older than our window
	cutoff := now.Add(-l.duration)
	newTimestamps := []time.Time{}

	for _, ts := range timestamps {
		if ts.After(cutoff) {
			newTimestamps = append(newTimestamps, ts)
		}
	}

	// Check if we're over the limit
	if len(newTimestamps) >= l.limit {
		l.logger.WithFields(logrus.Fields{
			"key":      key,
			"requests": len(newTimestamps),
			"limit":    l.limit,
		}).Debug("Rate limit exceeded")
		return false
	}

	// Add current timestamp and update
	newTimestamps = append(newTimestamps, now)
	l.requests[key] = newTimestamps
	return true
}

// Cleanup removes expired timestamps
func (l *LocalLimiter) cleanup(now time.Time) {
	cutoff := now.Add(-l.duration)
	for key, timestamps := range l.requests {
		newTimestamps := []time.Time{}
		for _, ts := range timestamps {
			if ts.After(cutoff) {
				newTimestamps = append(newTimestamps, ts)
			}
		}

		if len(newTimestamps) == 0 {
			delete(l.requests, key)
		} else {
			l.requests[key] = newTimestamps
		}
	}
}

// RedisLimiter implements Redis-based rate limiting
type RedisLimiter struct {
	client   *redis.Client
	limit    int
	duration time.Duration
	logger   *logrus.Logger
}

// NewRedisLimiter creates a new Redis-based rate limiter
func NewRedisLimiter(redisURL string, limit int, duration time.Duration, logger *logrus.Logger) (*RedisLimiter, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &RedisLimiter{
		client:   client,
		limit:    limit,
		duration: duration,
		logger:   logger,
	}, nil
}

// Allow checks if a request is allowed based on rate limits
func (r *RedisLimiter) Allow(key string) bool {
	ctx := context.Background()
	redisKey := "ratelimit:" + key

	pipe := r.client.Pipeline()
	now := time.Now().UnixNano()
	cutoff := now - int64(r.duration)

	// Remove timestamps that are older than the window
	pipe.ZRemRangeByScore(ctx, redisKey, "-inf", fmt.Sprintf("%d", cutoff))
	// Count timestamps in the current window
	countCmd := pipe.ZCard(ctx, redisKey)
	// Add the current timestamp
	pipe.ZAdd(ctx, redisKey, redis.Z{Score: float64(now), Member: now})
	// Set key expiration
	pipe.Expire(ctx, redisKey, r.duration)

	_, err := pipe.Exec(ctx)
	if err != nil {
		r.logger.WithError(err).Error("Redis rate limit operation failed")
		return true // If Redis fails, allow the request
	}

	count := countCmd.Val()
	allowed := count < int64(r.limit)

	if !allowed {
		r.logger.WithFields(logrus.Fields{
			"key":      key,
			"requests": count,
			"limit":    r.limit,
		}).Debug("Rate limit exceeded")
	}

	return allowed
}

// New creates a rate limiter based on configuration
func New(config Config, logger *logrus.Logger) (RateLimiter, error) {
	if !config.Enabled {
		return &AlwaysAllowLimiter{}, nil
	}

	switch config.Strategy {
	case "redis":
		return NewRedisLimiter(config.RedisURL, config.Limit, config.Duration, logger)
	default:
		return NewLocalLimiter(config.Limit, config.Duration, logger), nil
	}
}

// AlwaysAllowLimiter is a no-op limiter that always allows requests
type AlwaysAllowLimiter struct{}

// Allow always returns true
func (l *AlwaysAllowLimiter) Allow(key string) bool {
	return true
}
