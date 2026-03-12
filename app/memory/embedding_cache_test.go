package memory

import (
	"testing"
	"time"
)

func TestEmbeddingCache_ExactHit(t *testing.T) {
	c := NewEmbeddingCache(10, time.Minute)
	c.Put("hello world", []float32{1, 2, 3})

	vec, ok := c.Get("hello world")
	if !ok || len(vec) != 3 {
		t.Fatal("expected cache hit")
	}
	if vec[0] != 1 || vec[1] != 2 || vec[2] != 3 {
		t.Fatalf("wrong vector: %v", vec)
	}
}

func TestEmbeddingCache_CaseInsensitive(t *testing.T) {
	c := NewEmbeddingCache(10, time.Minute)
	c.Put("Hello World", []float32{1, 0})

	vec, ok := c.Get("hello world")
	if !ok {
		t.Fatal("expected case-insensitive hit")
	}
	if vec[0] != 1 {
		t.Fatalf("wrong vector: %v", vec)
	}
}

func TestEmbeddingCache_Miss(t *testing.T) {
	c := NewEmbeddingCache(10, time.Minute)
	c.Put("hello", []float32{1})

	_, ok := c.Get("world")
	if ok {
		t.Fatal("expected miss")
	}
}

func TestEmbeddingCache_TTLExpiry(t *testing.T) {
	c := NewEmbeddingCache(10, 1*time.Millisecond)
	c.Put("hello", []float32{1})

	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get("hello")
	if ok {
		t.Fatal("expected expiry")
	}
}

func TestEmbeddingCache_LRUEviction(t *testing.T) {
	c := NewEmbeddingCache(2, time.Minute)
	c.Put("a", []float32{1})
	c.Put("b", []float32{2})
	c.Put("c", []float32{3}) // evicts "a"

	if c.Len() != 2 {
		t.Fatalf("expected 2 entries, got %d", c.Len())
	}

	_, ok := c.Get("a")
	if ok {
		t.Fatal("expected 'a' to be evicted")
	}
	_, ok = c.Get("b")
	if !ok {
		t.Fatal("expected 'b' to still exist")
	}
}

func TestEmbeddingCache_VectorCopied(t *testing.T) {
	c := NewEmbeddingCache(10, time.Minute)
	original := []float32{1, 2, 3}
	c.Put("test", original)

	// Mutate original
	original[0] = 99

	vec, _ := c.Get("test")
	if vec[0] != 1 {
		t.Fatal("cache should store a copy, not a reference")
	}

	// Mutate returned vector
	vec[1] = 99
	vec2, _ := c.Get("test")
	if vec2[1] != 2 {
		t.Fatal("cache should return a copy, not a reference")
	}
}

func TestEmbeddingCache_EmptyQuery(t *testing.T) {
	c := NewEmbeddingCache(10, time.Minute)
	c.Put("", []float32{1})
	if c.Len() != 0 {
		t.Fatal("empty query should not be cached")
	}

	_, ok := c.Get("")
	if ok {
		t.Fatal("empty query should miss")
	}
}

func TestEmbeddingCache_LRUTouchOnGet(t *testing.T) {
	c := NewEmbeddingCache(2, time.Minute)
	c.Put("a", []float32{1})
	c.Put("b", []float32{2})

	// Touch "a" so it becomes most-recently-used
	c.Get("a")

	// Insert "c" — should evict "b" (now LRU), not "a"
	c.Put("c", []float32{3})

	_, ok := c.Get("a")
	if !ok {
		t.Fatal("expected 'a' to survive (was touched)")
	}
	_, ok = c.Get("b")
	if ok {
		t.Fatal("expected 'b' to be evicted")
	}
}
