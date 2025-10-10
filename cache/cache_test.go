// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package cache

import (
	"context"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/pkg/types"
)

func TestMemoryCache_BasicOperations(t *testing.T) {
	ctx := context.Background()
	cache := NewMemoryCache(CacheConfig{
		MaxSize:        10,
		DefaultTTL:     1 * time.Minute,
		EvictionPolicy: EvictionPolicyLRU,
	})
	defer cache.Close()

	// Test Set and Get
	err := cache.Set(ctx, "key1", "value1", 1*time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, found := cache.Get(ctx, "key1")
	if !found {
		t.Fatal("Expected to find key1")
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}

	// Test Get non-existent key
	_, found = cache.Get(ctx, "nonexistent")
	if found {
		t.Error("Should not find nonexistent key")
	}

	// Test Delete
	err = cache.Delete(ctx, "key1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, found = cache.Get(ctx, "key1")
	if found {
		t.Error("Key should be deleted")
	}
}

func TestMemoryCache_TTLExpiration(t *testing.T) {
	ctx := context.Background()
	cache := NewMemoryCache(CacheConfig{
		MaxSize:        10,
		DefaultTTL:     50 * time.Millisecond,
		EvictionPolicy: EvictionPolicyLRU,
	})
	defer cache.Close()

	// Set with short TTL
	err := cache.Set(ctx, "key1", "value1", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should exist immediately
	_, found := cache.Get(ctx, "key1")
	if !found {
		t.Error("Key should exist")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, found = cache.Get(ctx, "key1")
	if found {
		t.Error("Key should be expired")
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	ctx := context.Background()
	cache := NewMemoryCache(CacheConfig{
		MaxSize:        10,
		DefaultTTL:     1 * time.Minute,
		EvictionPolicy: EvictionPolicyLRU,
	})
	defer cache.Close()

	// Add multiple entries
	cache.Set(ctx, "key1", "value1", 1*time.Minute)
	cache.Set(ctx, "key2", "value2", 1*time.Minute)
	cache.Set(ctx, "key3", "value3", 1*time.Minute)

	stats := cache.Stats()
	if stats.Size != 3 {
		t.Errorf("Expected size 3, got %d", stats.Size)
	}

	// Clear cache
	err := cache.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	stats = cache.Stats()
	if stats.Size != 0 {
		t.Errorf("Expected size 0 after clear, got %d", stats.Size)
	}

	_, found := cache.Get(ctx, "key1")
	if found {
		t.Error("Key should not exist after clear")
	}
}

func TestMemoryCache_LRUEviction(t *testing.T) {
	ctx := context.Background()
	cache := NewMemoryCache(CacheConfig{
		MaxSize:        3,
		DefaultTTL:     1 * time.Minute,
		EvictionPolicy: EvictionPolicyLRU,
	})
	defer cache.Close()

	// Fill cache
	cache.Set(ctx, "key1", "value1", 1*time.Minute)
	cache.Set(ctx, "key2", "value2", 1*time.Minute)
	cache.Set(ctx, "key3", "value3", 1*time.Minute)

	// Access key1 to make it recently used
	cache.Get(ctx, "key1")

	// Add new entry, should evict key2 (least recently used)
	cache.Set(ctx, "key4", "value4", 1*time.Minute)

	// key2 should be evicted
	_, found := cache.Get(ctx, "key2")
	if found {
		t.Error("key2 should be evicted")
	}

	// key1 should still exist
	_, found = cache.Get(ctx, "key1")
	if !found {
		t.Error("key1 should still exist")
	}
}

func TestMemoryCache_Stats(t *testing.T) {
	ctx := context.Background()
	cache := NewMemoryCache(CacheConfig{
		MaxSize:        10,
		DefaultTTL:     1 * time.Minute,
		EvictionPolicy: EvictionPolicyLRU,
		EnableMetrics:  true,
	})
	defer cache.Close()

	// Set some values
	cache.Set(ctx, "key1", "value1", 1*time.Minute)
	cache.Set(ctx, "key2", "value2", 1*time.Minute)

	// Generate hits
	cache.Get(ctx, "key1")
	cache.Get(ctx, "key1")

	// Generate miss
	cache.Get(ctx, "nonexistent")

	stats := cache.Stats()

	if stats.Sets != 2 {
		t.Errorf("Expected 2 sets, got %d", stats.Sets)
	}

	if stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	expectedHitRate := float64(2) / float64(3)
	if stats.HitRate < expectedHitRate-0.01 || stats.HitRate > expectedHitRate+0.01 {
		t.Errorf("Expected hit rate ~%.2f, got %.2f", expectedHitRate, stats.HitRate)
	}

	if stats.Size != 2 {
		t.Errorf("Expected size 2, got %d", stats.Size)
	}
}

func TestDefaultKeyFunc(t *testing.T) {
	msg1 := types.NewMessage(types.MessageRoleUser, []types.Part{
		types.NewTextPart("Hello"),
	})

	msg2 := types.NewMessage(types.MessageRoleUser, []types.Part{
		types.NewTextPart("Hello"),
	})

	msg3 := types.NewMessage(types.MessageRoleUser, []types.Part{
		types.NewTextPart("World"),
	})

	key1 := DefaultKeyFunc(msg1)
	key2 := DefaultKeyFunc(msg2)
	key3 := DefaultKeyFunc(msg3)

	// Same content should generate same key
	if key1 != key2 {
		t.Error("Same messages should generate same key")
	}

	// Different content should generate different key
	if key1 == key3 {
		t.Error("Different messages should generate different keys")
	}

	// Key should be hex string
	if len(key1) != 64 {
		t.Errorf("Expected 64 char hex string, got %d chars", len(key1))
	}
}

func TestResponseCache(t *testing.T) {
	ctx := context.Background()
	memCache := NewMemoryCache(DefaultCacheConfig())
	defer memCache.Close()

	responseCache := NewResponseCache(memCache, DefaultResponseCacheConfig())

	userMsg := types.NewMessage(types.MessageRoleUser, []types.Part{
		types.NewTextPart("Hello"),
	})

	responseMsg := types.NewMessage("assistant", []types.Part{
		types.NewTextPart("Hi there!"),
	})

	// Should not be cached initially
	_, found := responseCache.Get(ctx, userMsg)
	if found {
		t.Error("Should not find uncached message")
	}

	// Set cache
	err := responseCache.Set(ctx, userMsg, responseMsg)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should be cached now
	cachedResponse, found := responseCache.Get(ctx, userMsg)
	if !found {
		t.Error("Should find cached message")
	}

	if len(cachedResponse.Parts) != 1 {
		t.Fatalf("Expected 1 part, got %d", len(cachedResponse.Parts))
	}

	if textPart, ok := cachedResponse.Parts[0].(*types.TextPart); ok {
		if textPart.Text != "Hi there!" {
			t.Errorf("Expected 'Hi there!', got '%s'", textPart.Text)
		}
	} else {
		t.Error("Expected TextPart")
	}
}

func TestResponseCache_Disabled(t *testing.T) {
	ctx := context.Background()
	memCache := NewMemoryCache(DefaultCacheConfig())
	defer memCache.Close()

	config := DefaultResponseCacheConfig()
	config.Enabled = false
	responseCache := NewResponseCache(memCache, config)

	userMsg := types.NewMessage(types.MessageRoleUser, []types.Part{
		types.NewTextPart("Hello"),
	})

	responseMsg := types.NewMessage("assistant", []types.Part{
		types.NewTextPart("Hi!"),
	})

	// Set should not cache when disabled
	err := responseCache.Set(ctx, userMsg, responseMsg)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should not be cached
	_, found := responseCache.Get(ctx, userMsg)
	if found {
		t.Error("Should not cache when disabled")
	}
}

func TestMemoryCache_Concurrent(t *testing.T) {
	ctx := context.Background()
	cache := NewMemoryCache(CacheConfig{
		MaxSize:        100,
		DefaultTTL:     1 * time.Minute,
		EvictionPolicy: EvictionPolicyLRU,
	})
	defer cache.Close()

	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 10; j++ {
				cache.Set(ctx, "key", n, 1*time.Minute)
			}
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				cache.Get(ctx, "key")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Should not panic
	stats := cache.Stats()
	if stats.Sets == 0 {
		t.Error("Expected some sets")
	}
}
