package repository

import (
	"context"
	"testing"
	"time"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

func TestConversationRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewConversationRepository(db, zap.NewNop())
	ctx := context.Background()

	// 1. Create
	conv := &domain.Conversation{
		Title:   "Test Conversation",
		Source:  "telegram",
		Channel: "bot",
	}

	if err := repo.Create(ctx, conv); err != nil {
		t.Fatalf("failed to create: %v", err)
	}
	if conv.ID == 0 {
		t.Fatal("expected conversation ID to be set after insert")
	}

	// 2. FindByID
	found, err := repo.FindByID(ctx, conv.ID)
	if err != nil {
		t.Fatalf("failed to find by id: %v", err)
	}
	if found == nil || found.Title != "Test Conversation" {
		t.Fatal("conversation not found or invalid title")
	}

	// 3. Update
	conv.Title = "Updated Conversation"
	conv.UpdatedAt = time.Now()
	if err := repo.Update(ctx, conv); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	foundUpdated, _ := repo.FindByID(ctx, conv.ID)
	if foundUpdated.Title != "Updated Conversation" {
		t.Fatal("conversation title was not updated")
	}

	// 4. FindAll
	all, err := repo.FindAll(ctx)
	if err != nil || len(all) != 1 {
		t.Fatal("failed to find all or invalid length")
	}

	// 5. Delete
	if err := repo.Delete(ctx, conv.ID); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}
	deleted, _ := repo.FindByID(ctx, conv.ID)
	if deleted != nil {
		t.Fatal("conversation was not really deleted")
	}
}
