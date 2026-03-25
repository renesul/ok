package agent

import "testing"

func TestRateLimiter_WithinLimit(t *testing.T) {
	rl := NewRateLimiter()
	for i := 0; i < 5; i++ {
		if err := rl.Allow("shell"); err != nil {
			t.Errorf("call %d should be allowed: %v", i+1, err)
		}
	}
}

func TestRateLimiter_ExceedsLimit(t *testing.T) {
	rl := NewRateLimiter()
	// Shell limit is 5/min
	for i := 0; i < 5; i++ {
		rl.Allow("shell")
	}
	if err := rl.Allow("shell"); err == nil {
		t.Fatal("6th call should be blocked")
	}
}

func TestRateLimiter_UnlimitedTool(t *testing.T) {
	rl := NewRateLimiter()
	for i := 0; i < 100; i++ {
		if err := rl.Allow("echo"); err != nil {
			t.Errorf("unlimited tool should always pass: %v", err)
		}
	}
}

func TestRateLimiter_IndependentTools(t *testing.T) {
	rl := NewRateLimiter()
	// Exhaust shell limit
	for i := 0; i < 5; i++ {
		rl.Allow("shell")
	}
	// HTTP should still work (different counter)
	if err := rl.Allow("http"); err != nil {
		t.Errorf("http should be independent of shell: %v", err)
	}
}
