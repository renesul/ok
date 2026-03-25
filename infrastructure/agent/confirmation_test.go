package agent

import (
	"testing"
	"time"
)

func TestConfirmation_RequestAndApprove(t *testing.T) {
	cm := NewConfirmationManager()
	conf := cm.Request("shell", "rm -rf /tmp/test")

	go func() {
		time.Sleep(10 * time.Millisecond)
		cm.Respond(conf.ID, true)
	}()

	approved, err := cm.WaitForResponse(conf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true")
	}
}

func TestConfirmation_RequestAndDeny(t *testing.T) {
	cm := NewConfirmationManager()
	conf := cm.Request("shell", "sudo reboot")

	go func() {
		time.Sleep(10 * time.Millisecond)
		cm.Respond(conf.ID, false)
	}()

	approved, err := cm.WaitForResponse(conf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false")
	}
}

func TestConfirmation_RespondNotFound(t *testing.T) {
	cm := NewConfirmationManager()
	err := cm.Respond("nonexistent-id", true)
	if err == nil {
		t.Fatal("expected error for nonexistent confirmation")
	}
}
