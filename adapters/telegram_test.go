package adapters

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

func newTestTelegram(runner AgentRunner, ownerID int64) *TelegramAdapter {
	return NewTelegramAdapter(runner, "fake-token", ownerID, zap.NewNop())
}

func telegramMsg(fromID int64, chatType, text string) *tgbotapi.Message {
	msg := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{Type: chatType},
		Text: text,
	}
	if fromID != 0 {
		msg.From = &tgbotapi.User{ID: fromID}
	}
	return msg
}

// --- Enabled ---

func TestTelegramAdapter_Enabled(t *testing.T) {
	tests := []struct {
		token   string
		ownerID int64
		want    bool
	}{
		{"tok", 123, true},
		{"", 123, false},
		{"tok", 0, false},
		{"", 0, false},
	}
	for _, tt := range tests {
		a := NewTelegramAdapter(newMockRunner(), tt.token, tt.ownerID, zap.NewNop())
		if got := a.Enabled(); got != tt.want {
			t.Errorf("Enabled(%q, %d) = %v, want %v", tt.token, tt.ownerID, got, tt.want)
		}
	}
}

// --- handleMessage ---

func TestTelegramAdapter_HandleMessage_IgnoresGroup(t *testing.T) {
	mock := newMockRunner()
	a := newTestTelegram(mock, 123)
	a.handleMessage(telegramMsg(123, "group", "hi"))
	if mock.called {
		t.Error("expected Run() NOT called for group")
	}
}

func TestTelegramAdapter_HandleMessage_IgnoresSuperGroup(t *testing.T) {
	mock := newMockRunner()
	a := newTestTelegram(mock, 123)
	a.handleMessage(telegramMsg(123, "supergroup", "hi"))
	if mock.called {
		t.Error("expected Run() NOT called for supergroup")
	}
}

func TestTelegramAdapter_HandleMessage_IgnoresNonOwner(t *testing.T) {
	mock := newMockRunner()
	a := newTestTelegram(mock, 123)
	a.handleMessage(telegramMsg(999, "private", "hi"))
	if mock.called {
		t.Error("expected Run() NOT called for non-owner")
	}
}

func TestTelegramAdapter_HandleMessage_IgnoresEmptyText(t *testing.T) {
	mock := newMockRunner()
	a := newTestTelegram(mock, 123)
	a.handleMessage(telegramMsg(123, "private", ""))
	if mock.called {
		t.Error("expected Run() NOT called for empty text")
	}
}

func TestTelegramAdapter_HandleMessage_ProcessesOwner(t *testing.T) {
	mock := newMockRunner()
	a := newTestTelegram(mock, 123)
	a.handleMessage(telegramMsg(123, "private", "listar tarefas"))
	if !mock.called {
		t.Fatal("expected Run() to be called")
	}
	if mock.lastInput != "listar tarefas" {
		t.Errorf("Run() input = %q, want %q", mock.lastInput, "listar tarefas")
	}
}

func TestTelegramAdapter_HandleMessage_PanicRecovery(t *testing.T) {
	runner := newPanickingRunner("banco corrompido")
	a := newTestTelegram(runner, 123)
	a.handleMessage(telegramMsg(123, "private", "trigger panic"))
	if !runner.called {
		t.Fatal("expected Run() to be called before panic")
	}
}

func TestTelegramAdapter_HandleMessage_NilFrom(t *testing.T) {
	mock := newMockRunner()
	a := newTestTelegram(mock, 123)
	msg := telegramMsg(0, "private", "hi")
	msg.From = nil
	// Should not panic on nil From
	a.handleMessage(msg)
	if mock.called {
		t.Error("expected Run() NOT called for nil From")
	}
}
