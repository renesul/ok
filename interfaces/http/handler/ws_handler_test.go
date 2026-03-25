package handler

import (
	"testing"
)

func TestWSHub_HydrationState(t *testing.T) {
	hub := NewWSHub()

	hub.mu.Lock()
	hub.isRunning = true
	hub.currentPhase = "act"
	hub.terminalHistory = []string{"line1", "line2"}
	hub.mu.Unlock()

	state := hub.HydrationState()
	if state["running"] != true {
		t.Error("expected running=true")
	}
	if state["phase"] != "act" {
		t.Errorf("phase = %q, want 'act'", state["phase"])
	}
	history := state["terminal_history"].([]string)
	if len(history) != 2 {
		t.Errorf("terminal_history len = %d, want 2", len(history))
	}
}

func TestWSHub_HydrationState_Empty(t *testing.T) {
	hub := NewWSHub()

	state := hub.HydrationState()
	if state["running"] != false {
		t.Error("expected running=false by default")
	}
	if state["phase"] != "" {
		t.Errorf("phase = %q, want empty", state["phase"])
	}
}

func TestWSHub_TerminalHistoryCap(t *testing.T) {
	hub := NewWSHub()

	hub.mu.Lock()
	for i := 0; i < 150; i++ {
		hub.terminalHistory = append(hub.terminalHistory, "line")
	}
	if len(hub.terminalHistory) > 100 {
		hub.terminalHistory = hub.terminalHistory[len(hub.terminalHistory)-100:]
	}
	hub.mu.Unlock()

	hub.mu.RLock()
	count := len(hub.terminalHistory)
	hub.mu.RUnlock()

	if count != 100 {
		t.Errorf("terminal history = %d entries, want 100 (cap)", count)
	}
}

func TestWSHub_PhaseTracking(t *testing.T) {
	hub := NewWSHub()

	hub.mu.Lock()
	hub.currentPhase = "observe"
	hub.mu.Unlock()

	state := hub.HydrationState()
	if state["phase"] != "observe" {
		t.Errorf("phase = %q, want 'observe'", state["phase"])
	}

	hub.mu.Lock()
	hub.currentPhase = "act"
	hub.mu.Unlock()

	state = hub.HydrationState()
	if state["phase"] != "act" {
		t.Errorf("phase after update = %q, want 'act'", state["phase"])
	}
}

func TestWSHub_DoneResetsRunning(t *testing.T) {
	hub := NewWSHub()

	hub.mu.Lock()
	hub.isRunning = true
	hub.mu.Unlock()

	state := hub.HydrationState()
	if state["running"] != true {
		t.Error("expected running=true before done")
	}

	hub.mu.Lock()
	hub.isRunning = false
	hub.mu.Unlock()

	state = hub.HydrationState()
	if state["running"] != false {
		t.Error("expected running=false after done")
	}
}

func TestWSHub_InitialState(t *testing.T) {
	hub := NewWSHub()
	state := hub.HydrationState()

	if state["running"] != false {
		t.Error("initial running should be false")
	}
	if state["phase"] != "" {
		t.Errorf("initial phase should be empty, got %q", state["phase"])
	}
	if state["terminal_history"] != nil {
		history := state["terminal_history"].([]string)
		if len(history) != 0 {
			t.Errorf("initial terminal_history should be empty, got %d", len(history))
		}
	}
}
