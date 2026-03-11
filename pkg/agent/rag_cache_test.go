package agent

import (
	"testing"
	"time"
)

func TestRAGContextCache_ExactHit(t *testing.T) {
	c := NewRAGContextCache(10, time.Minute)
	c.Put("weather forecast", "## Relevant Past Interactions\n1. sunny day")

	got, ok := c.Get("weather forecast")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got != "## Relevant Past Interactions\n1. sunny day" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestRAGContextCache_CaseInsensitive(t *testing.T) {
	c := NewRAGContextCache(10, time.Minute)
	c.Put("Weather Forecast", "context-a")

	got, ok := c.Get("weather forecast")
	if !ok {
		t.Fatal("expected case-insensitive hit")
	}
	if got != "context-a" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestRAGContextCache_SimilarityHit(t *testing.T) {
	c := NewRAGContextCache(10, time.Minute)
	c.Put("what is the weather forecast today", "weather-context")

	// Similar query (high trigram overlap)
	got, ok := c.Get("what is the weather forecast for today")
	if !ok {
		t.Fatal("expected similarity hit")
	}
	if got != "weather-context" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestRAGContextCache_DissimilarMiss(t *testing.T) {
	c := NewRAGContextCache(10, time.Minute)
	c.Put("what is the weather today", "weather-context")

	_, ok := c.Get("how to program in go")
	if ok {
		t.Fatal("expected miss for dissimilar query")
	}
}

func TestRAGContextCache_TTLExpiry(t *testing.T) {
	c := NewRAGContextCache(10, 1*time.Millisecond)
	c.Put("hello", "context")

	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get("hello")
	if ok {
		t.Fatal("expected expiry")
	}
}

func TestRAGContextCache_LRUEviction(t *testing.T) {
	c := NewRAGContextCache(2, time.Minute)
	c.Put("query-a", "ctx-a")
	c.Put("query-b", "ctx-b")
	c.Put("query-c", "ctx-c") // evicts "query-a"

	if c.Len() != 2 {
		t.Fatalf("expected 2 entries, got %d", c.Len())
	}

	_, ok := c.Get("query-a")
	if ok {
		t.Fatal("expected 'query-a' to be evicted")
	}
}

func TestRAGContextCache_EmptyQuery(t *testing.T) {
	c := NewRAGContextCache(10, time.Minute)
	c.Put("", "ctx")
	if c.Len() != 0 {
		t.Fatal("empty query should not be cached")
	}
}

func TestRAGContextCache_Trigrams(t *testing.T) {
	trigrams := ragBuildTrigrams("hello")
	if len(trigrams) == 0 {
		t.Fatal("expected trigrams for 'hello'")
	}

	// Short strings
	if len(ragBuildTrigrams("ab")) != 0 {
		t.Fatal("expected no trigrams for 'ab'")
	}
}

func TestRAGContextCache_JaccardSimilarity(t *testing.T) {
	a := ragBuildTrigrams("hello world")
	b := ragBuildTrigrams("hello world")
	if ragJaccardSimilarity(a, b) != 1.0 {
		t.Fatal("identical strings should have similarity 1.0")
	}

	c := ragBuildTrigrams("completely different text")
	sim := ragJaccardSimilarity(a, c)
	if sim > 0.3 {
		t.Fatalf("dissimilar strings should have low similarity, got %f", sim)
	}
}
