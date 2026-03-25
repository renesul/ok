package repository

import (
	"context"
	"testing"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

func TestMessageRepository_CreateAndFind(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewMessageRepository(db, zap.NewNop())
	convRepo := NewConversationRepository(db, zap.NewNop())
	ctx := context.Background()

	// Need a Conversation first because of Foreign Key (if any)
	conv := &domain.Conversation{Title: "Msg Test"}
	convRepo.Create(ctx, conv)

	msg := &domain.Message{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        "Hello OK bot",
		SortOrder:      1,
	}

	if err := repo.Create(ctx, msg); err != nil {
		t.Fatalf("failed to create msg: %v", err)
	}

	msgs, err := repo.FindByConversationID(ctx, conv.ID)
	if err != nil || len(msgs) != 1 {
		t.Fatalf("failed to find msgs: %v", err)
	}

	count, _ := repo.CountByConversationID(ctx, conv.ID)
	if count != 1 {
		t.Fatal("wrong count")
	}
}

func TestMessageRepository_SearchFTS5(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewMessageRepository(db, zap.NewNop())
	convRepo := NewConversationRepository(db, zap.NewNop())
	ctx := context.Background()

	conv := &domain.Conversation{Title: "FTS5 Test"}
	convRepo.Create(ctx, conv)

	msg := &domain.Message{
		ConversationID: conv.ID,
		Role:           "agent",
		Content:        "Abacaxi azul voador",
	}
	repo.Create(ctx, msg)
	_ = repo.IndexForSearch(ctx, []domain.Message{*msg})

	// Search via ConvRepo
	results, err := convRepo.Search(ctx, "Abacaxi")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatal("FTS5 did not match 'Abacaxi'")
	}

	results2, _ := convRepo.Search(ctx, "Morango")
	if len(results2) != 0 {
		t.Fatal("FTS5 matched incorrectly")
	}
}

func TestMessageRepository_SaveAndFindEmbeddings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewMessageRepository(db, zap.NewNop())
	ctx := context.Background()

	convID := uint(100)
	msgID := uint(500)
	embedding := []float32{0.1, 0.2, -0.5, 3.14}

	err := repo.SaveEmbedding(ctx, msgID, convID, embedding)
	if err != nil {
		t.Fatalf("failed to save embedding: %v", err)
	}

	all, err := repo.FindAllEmbeddings(ctx)
	if err != nil {
		t.Fatalf("failed to find embeddings: %v", err)
	}
	if len(all) != 1 {
		t.Fatal("expected exactly 1 embedding result")
	}
	
	if len(all[0].Embedding) != 4 || all[0].Embedding[0] != 0.1 || all[0].Embedding[3] != 3.14 {
		t.Fatal("embedding decoding (bytesToFloat32) miscalculated floats")
	}
}
