// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

/*
Package cache provides caching functionality for SAGE ADK agents.

This package implements various caching strategies to improve performance
by storing and reusing expensive computation results.

Features:
  - Multiple cache backends (memory, Redis)
  - TTL-based expiration
  - LRU eviction policy
  - Cache key generation from messages
  - Distributed caching support
  - Cache invalidation strategies

Example:

	import "github.com/sage-x-project/sage-adk/cache"

	// Create cache
	cache := cache.NewMemoryCache(cache.CacheConfig{
	    MaxSize: 1000,
	    TTL:     5 * time.Minute,
	})

	// Set cache entry
	cache.Set(ctx, "key", response, 5*time.Minute)

	// Get cache entry
	if response, found := cache.Get(ctx, "key"); found {
	    // Use cached response
	}

	// Delete cache entry
	cache.Delete(ctx, "key")
*/
package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sage-x-project/sage-adk/pkg/types"
)

// Cache defines the interface for caching implementations
type Cache interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) (interface{}, bool)

	// Set stores a value in cache with TTL
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a value from cache
	Delete(ctx context.Context, key string) error

	// Clear removes all entries from cache
	Clear(ctx context.Context) error

	// Stats returns cache statistics
	Stats() CacheStats

	// Close closes the cache
	Close() error
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	// MaxSize is the maximum number of entries
	MaxSize int

	// DefaultTTL is the default time-to-live
	DefaultTTL time.Duration

	// EvictionPolicy determines how entries are evicted
	EvictionPolicy EvictionPolicy

	// EnableMetrics enables cache metrics collection
	EnableMetrics bool
}

// EvictionPolicy determines how cache entries are evicted
type EvictionPolicy string

const (
	// EvictionPolicyLRU evicts least recently used entries
	EvictionPolicyLRU EvictionPolicy = "lru"

	// EvictionPolicyLFU evicts least frequently used entries
	EvictionPolicyLFU EvictionPolicy = "lfu"

	// EvictionPolicyFIFO evicts oldest entries first
	EvictionPolicyFIFO EvictionPolicy = "fifo"

	// EvictionPolicyTTL evicts based on TTL only
	EvictionPolicyTTL EvictionPolicy = "ttl"
)

// CacheStats holds cache statistics
type CacheStats struct {
	Hits          int64
	Misses        int64
	Sets          int64
	Deletes       int64
	Evictions     int64
	Size          int
	MaxSize       int
	HitRate       float64
	MemoryUsageKB int64
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		MaxSize:        1000,
		DefaultTTL:     5 * time.Minute,
		EvictionPolicy: EvictionPolicyLRU,
		EnableMetrics:  true,
	}
}

// ResponseCache is a specialized cache for message responses
type ResponseCache struct {
	cache  Cache
	config ResponseCacheConfig
}

// ResponseCacheConfig holds response cache configuration
type ResponseCacheConfig struct {
	// Enabled enables/disables caching
	Enabled bool

	// TTL is the default cache TTL
	TTL time.Duration

	// KeyFunc generates cache keys from messages
	KeyFunc func(*types.Message) string

	// ShouldCache determines if a message should be cached
	ShouldCache func(*types.Message) bool

	// ShouldInvalidate determines if cache should be invalidated
	ShouldInvalidate func(*types.Message) bool
}

// DefaultResponseCacheConfig returns default response cache configuration
func DefaultResponseCacheConfig() ResponseCacheConfig {
	return ResponseCacheConfig{
		Enabled:          true,
		TTL:              5 * time.Minute,
		KeyFunc:          DefaultKeyFunc,
		ShouldCache:      DefaultShouldCache,
		ShouldInvalidate: DefaultShouldInvalidate,
	}
}

// NewResponseCache creates a new response cache
func NewResponseCache(cache Cache, config ResponseCacheConfig) *ResponseCache {
	return &ResponseCache{
		cache:  cache,
		config: config,
	}
}

// Get retrieves a cached response
func (rc *ResponseCache) Get(ctx context.Context, msg *types.Message) (*types.Message, bool) {
	if !rc.config.Enabled {
		return nil, false
	}

	key := rc.config.KeyFunc(msg)
	value, found := rc.cache.Get(ctx, key)
	if !found {
		return nil, false
	}

	response, ok := value.(*types.Message)
	if !ok {
		return nil, false
	}

	return response, true
}

// Set stores a response in cache
func (rc *ResponseCache) Set(ctx context.Context, msg *types.Message, response *types.Message) error {
	if !rc.config.Enabled {
		return nil
	}

	if !rc.config.ShouldCache(msg) {
		return nil
	}

	key := rc.config.KeyFunc(msg)
	return rc.cache.Set(ctx, key, response, rc.config.TTL)
}

// Invalidate removes a cached response
func (rc *ResponseCache) Invalidate(ctx context.Context, msg *types.Message) error {
	if !rc.config.Enabled {
		return nil
	}

	if !rc.config.ShouldInvalidate(msg) {
		return nil
	}

	key := rc.config.KeyFunc(msg)
	return rc.cache.Delete(ctx, key)
}

// Stats returns cache statistics
func (rc *ResponseCache) Stats() CacheStats {
	return rc.cache.Stats()
}

// DefaultKeyFunc generates a cache key from a message
func DefaultKeyFunc(msg *types.Message) string {
	// Create a deterministic key from message content
	data := make(map[string]interface{})

	// Include role
	data["role"] = string(msg.Role)

	// Include parts
	parts := make([]string, len(msg.Parts))
	for i, part := range msg.Parts {
		if textPart, ok := part.(*types.TextPart); ok {
			parts[i] = textPart.Text
		}
	}
	data["parts"] = parts

	// Include context if present
	if msg.ContextID != nil {
		data["context"] = *msg.ContextID
	}

	// Marshal to JSON
	jsonData, _ := json.Marshal(data)

	// Generate SHA-256 hash
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// DefaultShouldCache determines if a message should be cached
func DefaultShouldCache(msg *types.Message) bool {
	// Cache user messages by default
	return msg.Role == types.MessageRoleUser
}

// DefaultShouldInvalidate determines if cache should be invalidated
func DefaultShouldInvalidate(msg *types.Message) bool {
	// Don't invalidate by default
	return false
}

// InvalidatePattern invalidates cache entries matching a pattern
func (rc *ResponseCache) InvalidatePattern(ctx context.Context, pattern string) error {
	// This would require cache backend support
	// For now, just clear all
	return rc.cache.Clear(ctx)
}

// CacheMiddleware creates a middleware that caches responses
func CacheMiddleware(cache *ResponseCache) func(next func(context.Context, *types.Message) (*types.Message, error)) func(context.Context, *types.Message) (*types.Message, error) {
	return func(next func(context.Context, *types.Message) (*types.Message, error)) func(context.Context, *types.Message) (*types.Message, error) {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			// Try to get from cache
			if response, found := cache.Get(ctx, msg); found {
				return response, nil
			}

			// Call next handler
			response, err := next(ctx, msg)
			if err != nil {
				return nil, err
			}

			// Store in cache
			cache.Set(ctx, msg, response)

			return response, nil
		}
	}
}

// warmupCache pre-populates cache with common queries
func (rc *ResponseCache) Warmup(ctx context.Context, queries []*types.Message) error {
	// This would be implemented to pre-populate cache
	return fmt.Errorf("not implemented")
}
