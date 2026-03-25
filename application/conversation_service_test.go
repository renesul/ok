package application

import (
	"context"
	"errors"
	"testing"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

func newTestConvService(convRepo *mockConversationRepo, msgRepo *mockMessageRepo) *ConversationService {
	return NewConversationService(convRepo, msgRepo, nil, zap.NewNop())
}

func TestConversationService_List(t *testing.T) {
	convRepo := &mockConversationRepo{
		findAllResult: []domain.Conversation{{ID: 1, Title: "test"}},
	}
	svc := newTestConvService(convRepo, &mockMessageRepo{})

	result, err := svc.ListConversations(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 || result[0].Title != "test" {
		t.Errorf("expected 1 conversation titled 'test', got %v", result)
	}
}

func TestConversationService_Get_Found(t *testing.T) {
	convRepo := &mockConversationRepo{
		findByIDResult: &domain.Conversation{ID: 1, Title: "found"},
	}
	svc := newTestConvService(convRepo, &mockMessageRepo{})

	conv, err := svc.GetConversation(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conv.Title != "found" {
		t.Errorf("title = %q, want 'found'", conv.Title)
	}
}

func TestConversationService_Get_NotFound(t *testing.T) {
	convRepo := &mockConversationRepo{findByIDResult: nil}
	svc := newTestConvService(convRepo, &mockMessageRepo{})

	_, err := svc.GetConversation(context.Background(), 999)
	if !errors.Is(err, ErrConversationNotFound) {
		t.Errorf("expected ErrConversationNotFound, got %v", err)
	}
}

func TestConversationService_Delete_CascadeOrder(t *testing.T) {
	convRepo := &mockConversationRepo{
		findByIDResult: &domain.Conversation{ID: 1},
	}
	msgRepo := &mockMessageRepo{}
	svc := newTestConvService(convRepo, msgRepo)

	err := svc.DeleteConversation(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify cascade: embeddings, FTS, messages, conversation
	if !msgRepo.deleteEmbCalled {
		t.Error("expected DeleteEmbeddings called")
	}
	if !msgRepo.deleteFTSCalled {
		t.Error("expected DeleteSearchIndex called")
	}
	if !msgRepo.deleteMsgCalled {
		t.Error("expected DeleteByConversationID called")
	}
	if !convRepo.deleteCalled {
		t.Error("expected conversation Delete called")
	}
}

func TestConversationService_Delete_NotFound(t *testing.T) {
	convRepo := &mockConversationRepo{findByIDResult: nil}
	svc := newTestConvService(convRepo, &mockMessageRepo{})

	err := svc.DeleteConversation(context.Background(), 999)
	if !errors.Is(err, ErrConversationNotFound) {
		t.Errorf("expected ErrConversationNotFound, got %v", err)
	}
}

func TestConversationService_Search_FTSFallback(t *testing.T) {
	convRepo := &mockConversationRepo{
		searchResult: []domain.Conversation{{ID: 1, Title: "fts result"}},
	}
	// embeddingService nil → goes straight to FTS
	svc := newTestConvService(convRepo, &mockMessageRepo{})

	result, err := svc.SearchConversations(context.Background(), "query")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 || result[0].Title != "fts result" {
		t.Errorf("expected FTS result, got %v", result)
	}
}
