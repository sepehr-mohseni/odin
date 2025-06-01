package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"odin/pkg/config"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
)

type CachedResponse struct {
	Headers    http.Header `json:"headers"`
	StatusCode int         `json:"status_code"`
	Body       []byte      `json:"body"`
}

type Store interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear()
	Close() error
}

type LocalStore struct {
	cache *cache.Cache
}

type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewStore(config config.CacheConfig) (Store, error) {
	switch config.Strategy {
	case "local":
		return &LocalStore{
			cache: cache.New(config.TTL, config.TTL*2),
		}, nil
	case "redis":
		opts, err := redis.ParseURL(config.RedisURL)
		if err != nil {
			return nil, fmt.Errorf("invalid Redis URL: %w", err)
		}
		client := redis.NewClient(opts)
		if err := client.Ping(context.Background()).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to Redis: %w", err)
		}
		return &RedisStore{
			client: client,
			ttl:    config.TTL,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported cache strategy: %s", config.Strategy)
	}
}

func (s *LocalStore) Get(key string) (interface{}, bool) {
	if value, found := s.cache.Get(key); found {
		return value, true
	}
	return nil, false
}

func (s *LocalStore) Set(key string, value interface{}, ttl time.Duration) {
	s.cache.Set(key, value, cache.DefaultExpiration)
}

func (s *LocalStore) Delete(key string) {
	s.cache.Delete(key)
}

func (s *LocalStore) Clear() {
	s.cache.Flush()
}

func (s *LocalStore) Close() error {
	return nil
}

func (s *RedisStore) Get(key string) (interface{}, bool) {
	ctx := context.Background()
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}

	var response CachedResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, false
	}

	return &response, true
}

func (s *RedisStore) Set(key string, value interface{}, ttl time.Duration) {
	ctx := context.Background()
	data, err := json.Marshal(value)
	if err != nil {
		return
	}

	if ttl > 0 {
		s.client.Set(ctx, key, data, ttl)
	} else {
		s.client.Set(ctx, key, data, s.ttl)
	}
}

func (s *RedisStore) Delete(key string) {
	ctx := context.Background()
	s.client.Del(ctx, key)
}

func (s *RedisStore) Clear() {
	ctx := context.Background()
	iter := s.client.Scan(ctx, 0, "cache:*", 100).Iterator()
	for iter.Next(ctx) {
		s.client.Del(ctx, iter.Val())
	}
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}
