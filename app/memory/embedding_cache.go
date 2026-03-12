// OK - Lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OK contributors

package memory

import (
	"strings"
	"sync"
	"time"
)

// EmbeddingCache caches query→vector mappings to avoid redundant embedding API calls.
// Thread-safe, TTL-based with LRU eviction. Exact string match only
// (embeddings are deterministic for the same model+input).
type EmbeddingCache struct {
	mu         sync.RWMutex
	entries    map[string]*embCacheEntry
	order      []string // LRU: oldest first
	maxEntries int
	ttl        time.Duration
}

type embCacheEntry struct {
	vector    []float32
	createdAt time.Time
}

// NewEmbeddingCache creates a cache for embedding vectors.
func NewEmbeddingCache(maxEntries int, ttl time.Duration) *EmbeddingCache {
	if maxEntries <= 0 {
		maxEntries = 100
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &EmbeddingCache{
		entries:    make(map[string]*embCacheEntry),
		order:      make([]string, 0),
		maxEntries: maxEntries,
		ttl:        ttl,
	}
}

// Get returns a cached embedding vector, or nil/false on miss.
func (c *EmbeddingCache) Get(query string) ([]float32, bool) {
	key := normalizeKey(query)
	if key == "" {
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok || time.Since(entry.createdAt) >= c.ttl {
		return nil, false
	}

	c.moveToEndLocked(key)
	return copyVector(entry.vector), true
}

// Put stores an embedding vector for a query.
func (c *EmbeddingCache) Put(query string, vector []float32) {
	key := normalizeKey(query)
	if key == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.evictExpiredLocked()

	if _, ok := c.entries[key]; ok {
		c.entries[key] = &embCacheEntry{
			vector:    copyVector(vector),
			createdAt: time.Now(),
		}
		c.moveToEndLocked(key)
		return
	}

	for len(c.entries) >= c.maxEntries && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, oldest)
	}

	c.entries[key] = &embCacheEntry{
		vector:    copyVector(vector),
		createdAt: time.Now(),
	}
	c.order = append(c.order, key)
}

// Len returns the number of entries.
func (c *EmbeddingCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

func (c *EmbeddingCache) evictExpiredLocked() {
	now := time.Now()
	newOrder := make([]string, 0, len(c.order))
	for _, key := range c.order {
		entry, ok := c.entries[key]
		if !ok || now.Sub(entry.createdAt) >= c.ttl {
			delete(c.entries, key)
			continue
		}
		newOrder = append(newOrder, key)
	}
	c.order = newOrder
}

func (c *EmbeddingCache) moveToEndLocked(key string) {
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
	c.order = append(c.order, key)
}

func normalizeKey(q string) string {
	return strings.ToLower(strings.TrimSpace(q))
}

func copyVector(v []float32) []float32 {
	cp := make([]float32, len(v))
	copy(cp, v)
	return cp
}
