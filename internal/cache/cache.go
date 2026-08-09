package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"Post_Analyzer_Webserver/config"
	"Post_Analyzer_Webserver/internal/logger"

	"github.com/redis/go-redis/v9"
)

// Cache defines the caching interface
type Cache interface {
	Get(ctx context.Context, key string, value interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

// MemoryCache implements an in-memory cache. Safe for concurrent use —
// every access to data goes through mu, since a Cache is shared across
// every concurrent HTTP request's goroutine plus the background cleanup
// goroutine.
type MemoryCache struct {
	mu   sync.RWMutex
	data map[string]cacheEntry
}

type cacheEntry struct {
	value      []byte
	expiration time.Time
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		data: make(map[string]cacheEntry),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a value from the cache
func (c *MemoryCache) Get(ctx context.Context, key string, value interface{}) error {
	c.mu.RLock()
	entry, exists := c.data[key]
	c.mu.RUnlock()
	if !exists {
		return fmt.Errorf("cache miss")
	}

	// Check expiration
	if time.Now().After(entry.expiration) {
		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()
		return fmt.Errorf("cache expired")
	}

	return json.Unmarshal(entry.value, value)
}

// Set stores a value in the cache
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.data[key] = cacheEntry{
		value:      data,
		expiration: time.Now().Add(ttl),
	}
	c.mu.Unlock()

	return nil
}

// Delete removes a value from the cache
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()
	return nil
}

// Clear removes all values from the cache
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	c.data = make(map[string]cacheEntry)
	c.mu.Unlock()
	return nil
}

// cleanup removes expired entries
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for key, entry := range c.data {
			if now.After(entry.expiration) {
				delete(c.data, key)
			}
		}
		c.mu.Unlock()
	}
}

// RedisCache implements Cache backed by Redis. Values are JSON-encoded,
// matching MemoryCache's wire format, so callers can't tell them apart.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache connects to Redis and verifies the connection with a PING.
func NewRedisCache(cfg *config.RedisConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &RedisCache{client: client}, nil
}

func (c *RedisCache) Get(ctx context.Context, key string, value interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("cache miss")
		}
		return err
	}
	return json.Unmarshal(data, value)
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *RedisCache) Clear(ctx context.Context) error {
	return c.client.FlushDB(ctx).Err()
}

// NewCache creates a cache based on configuration: Redis when enabled and
// reachable, falling back to an in-memory cache otherwise so a missing
// Redis instance degrades performance rather than availability.
func NewCache(cfg *config.Config) Cache {
	if cfg.Redis.Enabled {
		redisCache, err := NewRedisCache(&cfg.Redis)
		if err == nil {
			logger.Info("cache initialized", "type", "redis", "addr", cfg.Redis.Addr)
			return redisCache
		}
		logger.Warn("redis unavailable, falling back to in-memory cache", "addr", cfg.Redis.Addr, "error", err)
	}
	logger.Info("cache initialized", "type", "memory")
	return NewMemoryCache()
}
