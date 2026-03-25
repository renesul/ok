package adapters

import (
	"testing"

	"github.com/renesul/ok/domain"
)

func TestNormalizeResponse_Empty(t *testing.T) {
	got := NormalizeResponse(domain.AgentResponse{})
	if got != "" {
		t.Errorf("NormalizeResponse(empty) = %q, want empty", got)
	}
}

func TestNormalizeResponse_Messages(t *testing.T) {
	resp := domain.AgentResponse{
		Messages: []string{"hello", "world"},
	}
	got := NormalizeResponse(resp)
	want := "hello\nworld"
	if got != want {
		t.Errorf("NormalizeResponse(messages) = %q, want %q", got, want)
	}
}

func TestNormalizeResponse_StepsAndMessages(t *testing.T) {
	resp := domain.AgentResponse{
		Steps: []domain.StepResult{
			{Tool: "shell", Name: "build", Status: "done"},
		},
		Messages: []string{"ok"},
	}
	got := NormalizeResponse(resp)
	want := "[shell] build → done\nok"
	if got != want {
		t.Errorf("NormalizeResponse(steps+msgs) = %q, want %q", got, want)
	}
}
