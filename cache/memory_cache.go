// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package cache

import (
	"container/list"
	"context"
	"sync"
	"time"
)

// MemoryCache implements an in-memory LRU cache
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	lru     *list.List
	config  CacheConfig
	stats   CacheStats
}

type cacheEntry struct {
	key       string
	value     interface{}
	expiresAt time.Time
	element   *list.Element
	accessCount int64
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(config CacheConfig) *MemoryCache {
	if config.MaxSize == 0 {
		config = DefaultCacheConfig()
	}

	return &MemoryCache{
		entries: make(map[string]*cacheEntry),
		lru:     list.New(),
		config:  config,
		stats: CacheStats{
			MaxSize: config.MaxSize,
		},
	}
}

// Get retrieves a value from cache
func (c *MemoryCache) Get(ctx context.Context, key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, found := c.entries[key]
	if !found {
		c.stats.Misses++
		c.updateHitRate()
		return nil, false
	}

	// Check expiration
	if time.Now().After(entry.expiresAt) {
		c.deleteEntry(key)
		c.stats.Misses++
		c.updateHitRate()
		return nil, false
	}

	// Update LRU
	if c.config.EvictionPolicy == EvictionPolicyLRU {
		c.lru.MoveToFront(entry.element)
	}

	// Update access count for LFU
	if c.config.EvictionPolicy == EvictionPolicyLFU {
		entry.accessCount++
	}

	c.stats.Hits++
	c.updateHitRate()

	return entry.value, true
}

// Set stores a value in cache
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl == 0 {
		ttl = c.config.DefaultTTL
	}

	// Check if entry exists
	if entry, found := c.entries[key]; found {
		// Update existing entry
		entry.value = value
		entry.expiresAt = time.Now().Add(ttl)
		if c.config.EvictionPolicy == EvictionPolicyLRU {
			c.lru.MoveToFront(entry.element)
		}
		c.stats.Sets++
		return nil
	}

	// Evict if necessary
	if len(c.entries) >= c.config.MaxSize {
		c.evict()
	}

	// Create new entry
	entry := &cacheEntry{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(ttl),
		accessCount: 0,
	}

	// Add to LRU list
	entry.element = c.lru.PushFront(key)

	// Add to map
	c.entries[key] = entry
	c.stats.Sets++
	c.stats.Size = len(c.entries)

	return nil
}

// Delete removes a value from cache
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.deleteEntry(key)
	c.stats.Deletes++
	return nil
}

// Clear removes all entries from cache
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry)
	c.lru = list.New()
	c.stats.Size = 0

	return nil
}

// Stats returns cache statistics
func (c *MemoryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.stats
}

// Close closes the cache
func (c *MemoryCache) Close() error {
	return c.Clear(context.Background())
}

// deleteEntry removes an entry (must be called with lock held)
func (c *MemoryCache) deleteEntry(key string) {
	if entry, found := c.entries[key]; found {
		c.lru.Remove(entry.element)
		delete(c.entries, key)
		c.stats.Size = len(c.entries)
	}
}

// evict removes entries according to eviction policy
func (c *MemoryCache) evict() {
	switch c.config.EvictionPolicy {
	case EvictionPolicyLRU:
		c.evictLRU()
	case EvictionPolicyLFU:
		c.evictLFU()
	case EvictionPolicyFIFO:
		c.evictFIFO()
	case EvictionPolicyTTL:
		c.evictExpired()
	default:
		c.evictLRU()
	}
}

// evictLRU evicts least recently used entry
func (c *MemoryCache) evictLRU() {
	if element := c.lru.Back(); element != nil {
		key := element.Value.(string)
		c.deleteEntry(key)
		c.stats.Evictions++
	}
}

// evictLFU evicts least frequently used entry
func (c *MemoryCache) evictLFU() {
	var minAccess int64 = -1
	var victimKey string

	for key, entry := range c.entries {
		if minAccess == -1 || entry.accessCount < minAccess {
			minAccess = entry.accessCount
			victimKey = key
		}
	}

	if victimKey != "" {
		c.deleteEntry(victimKey)
		c.stats.Evictions++
	}
}

// evictFIFO evicts oldest entry
func (c *MemoryCache) evictFIFO() {
	if element := c.lru.Back(); element != nil {
		key := element.Value.(string)
		c.deleteEntry(key)
		c.stats.Evictions++
	}
}

// evictExpired removes all expired entries
func (c *MemoryCache) evictExpired() {
	now := time.Now()
	keysToDelete := make([]string, 0)

	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		c.deleteEntry(key)
		c.stats.Evictions++
	}
}

// updateHitRate calculates cache hit rate
func (c *MemoryCache) updateHitRate() {
	total := c.stats.Hits + c.stats.Misses
	if total > 0 {
		c.stats.HitRate = float64(c.stats.Hits) / float64(total)
	}
}

// CleanupExpired periodically removes expired entries
func (c *MemoryCache) CleanupExpired(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.mu.Lock()
			c.evictExpired()
			c.mu.Unlock()
		}
	}
}
