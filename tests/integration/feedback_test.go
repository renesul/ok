package integration

import (
	"testing"

	"github.com/renesul/ok/domain"
	agent "github.com/renesul/ok/infrastructure/agent"
	"go.uber.org/zap"
)

func TestFeedbackSave(t *testing.T) {
	defer cleanupFeedback(t)

	repo := agent.NewFeedbackRepository(testDB, zap.NewNop())

	err := repo.Save(nil, &domain.Feedback{
		ToolName:   "echo",
		Success:    true,
		DurationMs: 5,
	})
	if err != nil {
		t.Fatalf("save feedback failed: %v", err)
	}
}


