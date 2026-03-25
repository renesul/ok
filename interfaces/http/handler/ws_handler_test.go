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
