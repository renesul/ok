package application

import (
	"context"
	"testing"
	"time"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

func TestSessionService_ValidateSession_Valid(t *testing.T) {
	repo := &mockSessionRepo{
		findResult: &domain.Session{
			ID:        "valid-id",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		},
	}
	svc := NewSessionService(repo, zap.NewNop())

	valid, err := svc.ValidateSession(context.Background(), "valid-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("expected session to be valid")
	}
}

func TestSessionService_ValidateSession_Expired(t *testing.T) {
	repo := &mockSessionRepo{
		findResult: &domain.Session{
			ID:        "expired-id",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		},
	}
	svc := NewSessionService(repo, zap.NewNop())

	valid, err := svc.ValidateSession(context.Background(), "expired-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected session to be invalid (expired)")
	}
	if !repo.deleteCalled {
		t.Error("expected expired session to be deleted")
	}
}

func TestSessionService_ValidateSession_NotFound(t *testing.T) {
	repo := &mockSessionRepo{findResult: nil}
	svc := NewSessionService(repo, zap.NewNop())

	valid, err := svc.ValidateSession(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected session to be invalid (not found)")
	}
}
