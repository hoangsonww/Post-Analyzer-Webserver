package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"Post_Analyzer_Webserver/config"
)

// Cache defines the caching interface
type Cache interface {
	Get(ctx context.Context, key string, value interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

// MemoryCache implements an in-memory cache
type MemoryCache struct {
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
	entry, exists := c.data[key]
	if !exists {
		return fmt.Errorf("cache miss")
	}

	// Check expiration
	if time.Now().After(entry.expiration) {
		delete(c.data, key)
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

	c.data[key] = cacheEntry{
		value:      data,
		expiration: time.Now().Add(ttl),
	}

	return nil
}

// Delete removes a value from the cache
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	delete(c.data, key)
	return nil
}

// Clear removes all values from the cache
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.data = make(map[string]cacheEntry)
	return nil
}

// cleanup removes expired entries
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		for key, entry := range c.data {
			if now.After(entry.expiration) {
				delete(c.data, key)
			}
		}
	}
}

// NewCache creates a cache based on configuration
func NewCache(cfg *config.Config) Cache {
	// For now, always return memory cache
	// In the future, this could return Redis cache if configured
	return NewMemoryCache()
}
