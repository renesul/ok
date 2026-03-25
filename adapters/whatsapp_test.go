package adapters

import (
	"fmt"
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

func newTestWhatsApp(runner AgentRunner, owner string) *WhatsAppAdapter {
	return NewWhatsAppAdapter(runner, owner, "/tmp/test.db", zap.NewNop())
}

func whatsappMsg(sender string, isGroup bool, msg *waE2E.Message) *events.Message {
	return &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Sender:  types.JID{User: sender},
				IsGroup: isGroup,
			},
		},
		Message: msg,
	}
}

// --- Enabled ---

func TestWhatsAppAdapter_Enabled(t *testing.T) {
	tests := []struct {
		owner, dbPath string
		want          bool
	}{
		{"5511999", "/tmp/wa.db", true},
		{"", "/tmp/wa.db", false},
		{"5511999", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		a := NewWhatsAppAdapter(newMockRunner(), tt.owner, tt.dbPath, zap.NewNop())
		if got := a.Enabled(); got != tt.want {
			t.Errorf("Enabled(%q, %q) = %v, want %v", tt.owner, tt.dbPath, got, tt.want)
		}
	}
}

// --- extractText ---

func TestWhatsAppAdapter_ExtractText_Conversation(t *testing.T) {
	msg := &waE2E.Message{Conversation: strPtr("hello")}
	if got := extractText(msg); got != "hello" {
		t.Errorf("extractText(Conversation) = %q, want %q", got, "hello")
	}
}

func TestWhatsAppAdapter_ExtractText_ExtendedText(t *testing.T) {
	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{Text: strPtr("quoted")},
	}
	if got := extractText(msg); got != "quoted" {
		t.Errorf("extractText(ExtendedText) = %q, want %q", got, "quoted")
	}
}

func TestWhatsAppAdapter_ExtractText_Nil(t *testing.T) {
	if got := extractText(nil); got != "" {
		t.Errorf("extractText(nil) = %q, want empty", got)
	}
}

func TestWhatsAppAdapter_ExtractText_Empty(t *testing.T) {
	msg := &waE2E.Message{}
	if got := extractText(msg); got != "" {
		t.Errorf("extractText(empty) = %q, want empty", got)
	}
}

// --- handleMessage ---

func TestWhatsAppAdapter_HandleMessage_IgnoresGroup(t *testing.T) {
	mock := newMockRunner()
	a := newTestWhatsApp(mock, "5511999")
	a.handleMessage(whatsappMsg("5511999", true, &waE2E.Message{Conversation: strPtr("hi")}))
	if mock.called {
		t.Error("expected Run() NOT called for group message")
	}
}

func TestWhatsAppAdapter_HandleMessage_IgnoresNonOwner(t *testing.T) {
	mock := newMockRunner()
	a := newTestWhatsApp(mock, "5511999")
	a.handleMessage(whatsappMsg("5511888", false, &waE2E.Message{Conversation: strPtr("hi")}))
	if mock.called {
		t.Error("expected Run() NOT called for non-owner")
	}
}

func TestWhatsAppAdapter_HandleMessage_IgnoresEmptyText(t *testing.T) {
	mock := newMockRunner()
	a := newTestWhatsApp(mock, "5511999")
	a.handleMessage(whatsappMsg("5511999", false, &waE2E.Message{}))
	if mock.called {
		t.Error("expected Run() NOT called for empty text")
	}
}

func TestWhatsAppAdapter_HandleMessage_ProcessesOwner(t *testing.T) {
	mock := newMockRunner()
	a := newTestWhatsApp(mock, "5511999")
	a.handleMessage(whatsappMsg("5511999", false, &waE2E.Message{Conversation: strPtr("buscar clima")}))
	if !mock.called {
		t.Fatal("expected Run() to be called")
	}
	if mock.lastInput != "buscar clima" {
		t.Errorf("Run() input = %q, want %q", mock.lastInput, "buscar clima")
	}
}

func TestWhatsAppAdapter_HandleMessage_PanicRecovery(t *testing.T) {
	runner := newPanickingRunner("banco corrompido")
	a := newTestWhatsApp(runner, "5511999")
	// Should NOT panic the test — defer recover() in handleMessage catches it
	a.handleMessage(whatsappMsg("5511999", false, &waE2E.Message{Conversation: strPtr("trigger panic")}))
	if !runner.called {
		t.Fatal("expected Run() to be called before panic")
	}
}

func TestWhatsAppAdapter_HandleMessage_AgentError(t *testing.T) {
	mock := newMockRunner()
	mock.err = fmt.Errorf("agent internal error")
	a := newTestWhatsApp(mock, "5511999")
	// Should not crash on agent error
	a.handleMessage(whatsappMsg("5511999", false, &waE2E.Message{Conversation: strPtr("test error")}))
	if !mock.called {
		t.Fatal("expected Run() to be called")
	}
}
