// OK - Lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OK contributors

package context

import (
	"slices"
	"strings"
	"sync"
	"time"
)

// RAGContextCache caches formatted RAG context strings for queries.
// Uses trigram similarity so near-identical messages reuse cached RAG context
// without re-searching (avoids embedding API call + cosine search).
type RAGContextCache struct {
	mu         sync.RWMutex
	entries    map[string]*ragCacheEntry
	order      []string // LRU: oldest first
	maxEntries int
	ttl        time.Duration
}

type ragCacheEntry struct {
	key       string
	trigrams  []uint32
	context   string // formatted RAG context block
	createdAt time.Time
}

const ragSimilarityThreshold = 0.75

// NewRAGContextCache creates a cache for RAG context blocks.
func NewRAGContextCache(maxEntries int, ttl time.Duration) *RAGContextCache {
	if maxEntries <= 0 {
		maxEntries = 30
	}
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	return &RAGContextCache{
		entries:    make(map[string]*ragCacheEntry),
		order:      make([]string, 0),
		maxEntries: maxEntries,
		ttl:        ttl,
	}
}

// Get returns cached RAG context for an exact or similar query.
func (c *RAGContextCache) Get(query string) (string, bool) {
	normalized := ragNormalize(query)
	if normalized == "" {
		return "", false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Exact match
	if entry, ok := c.entries[normalized]; ok {
		if time.Since(entry.createdAt) < c.ttl {
			c.ragMoveToEndLocked(normalized)
			return entry.context, true
		}
	}

	// Trigram similarity match
	queryTrigrams := ragBuildTrigrams(normalized)
	var bestEntry *ragCacheEntry
	var bestSim float64

	for _, entry := range c.entries {
		if time.Since(entry.createdAt) >= c.ttl {
			continue
		}
		sim := ragJaccardSimilarity(queryTrigrams, entry.trigrams)
		if sim > bestSim {
			bestSim = sim
			bestEntry = entry
		}
	}

	if bestSim >= ragSimilarityThreshold && bestEntry != nil {
		c.ragMoveToEndLocked(bestEntry.key)
		return bestEntry.context, true
	}

	return "", false
}

// Put stores RAG context for a query.
func (c *RAGContextCache) Put(query string, ctx string) {
	normalized := ragNormalize(query)
	if normalized == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.ragEvictExpiredLocked()

	if _, ok := c.entries[normalized]; ok {
		c.entries[normalized] = &ragCacheEntry{
			key:       normalized,
			trigrams:  ragBuildTrigrams(normalized),
			context:   ctx,
			createdAt: time.Now(),
		}
		c.ragMoveToEndLocked(normalized)
		return
	}

	for len(c.entries) >= c.maxEntries && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, oldest)
	}

	c.entries[normalized] = &ragCacheEntry{
		key:       normalized,
		trigrams:  ragBuildTrigrams(normalized),
		context:   ctx,
		createdAt: time.Now(),
	}
	c.order = append(c.order, normalized)
}

// Len returns the number of entries.
func (c *RAGContextCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

func (c *RAGContextCache) ragEvictExpiredLocked() {
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

func (c *RAGContextCache) ragMoveToEndLocked(key string) {
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
	c.order = append(c.order, key)
}

func ragNormalize(q string) string {
	return strings.ToLower(strings.TrimSpace(q))
}

// ragBuildTrigrams generates sorted, deduplicated trigram hashes from a string.
func ragBuildTrigrams(s string) []uint32 {
	if len(s) < 3 {
		return nil
	}

	trigrams := make([]uint32, 0, len(s)-2)
	for i := 0; i <= len(s)-3; i++ {
		trigrams = append(trigrams, uint32(s[i])<<16|uint32(s[i+1])<<8|uint32(s[i+2]))
	}

	slices.Sort(trigrams)
	n := 1
	for i := 1; i < len(trigrams); i++ {
		if trigrams[i] != trigrams[i-1] {
			trigrams[n] = trigrams[i]
			n++
		}
	}
	return trigrams[:n]
}

// ragJaccardSimilarity computes |A ∩ B| / |A ∪ B| on sorted uint32 slices.
func ragJaccardSimilarity(a, b []uint32) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	i, j := 0, 0
	intersection := 0

	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			intersection++
			i++
			j++
		} else if a[i] < b[j] {
			i++
		} else {
			j++
		}
	}

	union := len(a) + len(b) - intersection
	return float64(intersection) / float64(union)
}
