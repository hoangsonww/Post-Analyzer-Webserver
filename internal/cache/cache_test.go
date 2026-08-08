package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"Post_Analyzer_Webserver/config"
)

func TestMemoryCache_SetAndGet(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()

	if err := c.Set(ctx, "key1", map[string]string{"hello": "world"}, time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	var got map[string]string
	if err := c.Get(ctx, "key1", &got); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got["hello"] != "world" {
		t.Errorf("unexpected value: %+v", got)
	}
}

func TestMemoryCache_Get_Miss(t *testing.T) {
	c := NewMemoryCache()
	var got string
	if err := c.Get(context.Background(), "nonexistent", &got); err == nil {
		t.Error("expected an error for a cache miss")
	}
}

func TestMemoryCache_Get_Expired(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()
	if err := c.Set(ctx, "key1", "value", -time.Second); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	var got string
	if err := c.Get(ctx, "key1", &got); err == nil {
		t.Error("expected an error reading an already-expired entry")
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()
	_ = c.Set(ctx, "key1", "value", time.Minute)

	if err := c.Delete(ctx, "key1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	var got string
	if err := c.Get(ctx, "key1", &got); err == nil {
		t.Error("expected the key to be gone after Delete")
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()
	_ = c.Set(ctx, "key1", "value", time.Minute)
	_ = c.Set(ctx, "key2", "value", time.Minute)

	if err := c.Clear(ctx); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	var got string
	if err := c.Get(ctx, "key1", &got); err == nil {
		t.Error("expected key1 to be gone after Clear")
	}
	if err := c.Get(ctx, "key2", &got); err == nil {
		t.Error("expected key2 to be gone after Clear")
	}
}

// TestMemoryCache_ConcurrentAccess exercises Get/Set/Delete from many
// goroutines at once — this is the normal access pattern in production
// (concurrent HTTP handlers sharing one Cache), so it must be race-free
// under `go test -race`.
func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func(i int) {
			defer wg.Done()
			_ = c.Set(ctx, "key", i, time.Minute)
		}(i)
		go func() {
			defer wg.Done()
			var v int
			_ = c.Get(ctx, "key", &v)
		}()
		go func() {
			defer wg.Done()
			_ = c.Delete(ctx, "key")
		}()
	}
	wg.Wait()
}

func TestNewCache_DisabledRedisFallsBackToMemory(t *testing.T) {
	cfg := &config.Config{Redis: config.RedisConfig{Enabled: false}}
	c := NewCache(cfg)
	if _, ok := c.(*MemoryCache); !ok {
		t.Errorf("expected NewCache with Redis disabled to return a *MemoryCache, got %T", c)
	}
}

func TestNewCache_UnreachableRedisFallsBackToMemory(t *testing.T) {
	cfg := &config.Config{Redis: config.RedisConfig{Enabled: true, Addr: "127.0.0.1:1"}}
	c := NewCache(cfg)
	if _, ok := c.(*MemoryCache); !ok {
		t.Errorf("expected NewCache with an unreachable Redis to fall back to *MemoryCache, got %T", c)
	}
}
