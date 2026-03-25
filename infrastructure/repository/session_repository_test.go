package repository

import (
	"context"
	"testing"
	"time"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

func TestSessionRepository_CreateAndFind(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSessionRepository(db, zap.NewNop())
	ctx := context.Background()

	sess := &domain.Session{
		ID:        "session_xyz",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := repo.Create(ctx, sess); err != nil {
		t.Fatalf("create error: %v", err)
	}

	found, err := repo.FindByID(ctx, "session_xyz")
	if err != nil || found == nil {
		t.Fatalf("failed to find session: %v", err)
	}
	if found.ID != "session_xyz" {
		t.Fatal("session ID mismatch")
	}

	if err := repo.DeleteByID(ctx, "session_xyz"); err != nil {
		t.Fatalf("delete error: %v", err)
	}
	deleted, _ := repo.FindByID(ctx, "session_xyz")
	if deleted != nil {
		t.Fatal("deleted session still found")
	}
}

func TestSessionRepository_DeleteExpired(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSessionRepository(db, zap.NewNop())
	ctx := context.Background()

	// 1. Valid session
	sessValid := &domain.Session{
		ID:        "valid_sess",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}
	repo.Create(ctx, sessValid)

	// 2. Expired session
	sessExpired := &domain.Session{
		ID:        "expired_sess",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now(),
	}
	repo.Create(ctx, sessExpired)

	if err := repo.DeleteExpired(ctx); err != nil {
		t.Fatalf("delete expired error: %v", err)
	}

	validFound, _ := repo.FindByID(ctx, "valid_sess")
	if validFound == nil {
		t.Fatal("valid session was incorrectly deleted")
	}

	expiredFound, _ := repo.FindByID(ctx, "expired_sess")
	if expiredFound != nil {
		t.Fatal("expired session was NOT deleted")
	}
}
