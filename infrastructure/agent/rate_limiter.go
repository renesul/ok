package agent

import (
	"fmt"
	"sync"
	"time"
)

// RateLimiter — controle de taxa por tool por minuto
type RateLimiter struct {
	mu       sync.Mutex
	counters map[string]int
	window   time.Time
	limits   map[string]int
}

// NewRateLimiter cria um rate limiter com limites por tool
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		counters: make(map[string]int),
		window:   time.Now(),
		limits: map[string]int{
			"shell":      5,
			"http":       20,
			"file_write": 10,
		},
	}
}

// Allow verifica se a tool pode ser executada dentro do limite
func (l *RateLimiter) Allow(tool string) error {
	limit, hasLimit := l.limits[tool]
	if !hasLimit {
		return nil // sem limite
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Reset window a cada minuto
	if time.Since(l.window) > time.Minute {
		l.counters = make(map[string]int)
		l.window = time.Now()
	}

	if l.counters[tool] >= limit {
		return fmt.Errorf("rate limit: %s excedeu %d chamadas/minuto", tool, limit)
	}

	l.counters[tool]++
	return nil
}
