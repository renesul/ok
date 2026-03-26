package application

import (
	"context"
	"errors"
	"testing"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

func newTestChatService(msgRepo *mockMessageRepo, convRepo *mockConversationRepo, llmConfigured bool) *ChatService {
	return NewChatService(convRepo, msgRepo, nil, nil, llmConfigured, zap.NewNop())
}

func TestChatService_BuildContext(t *testing.T) {
	msgs := make([]domain.Message, 60)
	for i := range msgs {
		msgs[i] = domain.Message{Role: "user", Content: "msg"}
	}

	msgRepo := &mockMessageRepo{findResult: msgs}
	svc := newTestChatService(msgRepo, &mockConversationRepo{}, true)

	result, err := svc.buildContext(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 50 {
		t.Errorf("expected 50 messages (maxContextMessages), got %d", len(result))
	}
}

func TestChatService_BuildContext_Empty(t *testing.T) {
	msgRepo := &mockMessageRepo{findResult: nil}
	svc := newTestChatService(msgRepo, &mockConversationRepo{}, true)

	result, err := svc.buildContext(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 messages, got %d", len(result))
	}
}

func TestChatService_BuildContext_FiltersRoles(t *testing.T) {
	msgs := []domain.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
		{Role: "tool", Content: "result"},
	}

	msgRepo := &mockMessageRepo{findResult: msgs}
	svc := newTestChatService(msgRepo, &mockConversationRepo{}, true)

	result, err := svc.buildContext(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 messages (user + assistant only), got %d", len(result))
	}
}

func TestChatService_SendMessage_NoLLM(t *testing.T) {
	svc := newTestChatService(&mockMessageRepo{}, &mockConversationRepo{}, false)

	_, err := svc.SendMessage(context.Background(), 1, "hello", nil)
	if !errors.Is(err, ErrLLMNotConfigured) {
		t.Errorf("expected ErrLLMNotConfigured, got %v", err)
	}
}

func TestChatService_CreateConversation_DefaultTitle(t *testing.T) {
	convRepo := &mockConversationRepo{}
	svc := newTestChatService(&mockMessageRepo{}, convRepo, true)

	conv, err := svc.CreateConversation(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conv.Title != "New conversation" {
		t.Errorf("title = %q, want 'New conversation'", conv.Title)
	}
	if conv.Source != "chat" {
		t.Errorf("source = %q, want 'chat'", conv.Source)
	}
}
