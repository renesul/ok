package tools

import (
	"strings"
	"testing"

	"github.com/renesul/ok/domain"
)

type mockSkillRepo struct {
	skills map[string]*domain.Skill
}

func (m *mockSkillRepo) List() ([]domain.Skill, error) {
	var result []domain.Skill
	for _, s := range m.skills {
		result = append(result, *s)
	}
	return result, nil
}

func (m *mockSkillRepo) Get(name string) (*domain.Skill, error) {
	s, ok := m.skills[name]
	if !ok {
		return nil, &skillNotFoundError{name: name}
	}
	return s, nil
}

type skillNotFoundError struct{ name string }

func (e *skillNotFoundError) Error() string { return "skill '" + e.name + "' not found" }

func newMockSkillRepo() *mockSkillRepo {
	return &mockSkillRepo{
		skills: map[string]*domain.Skill{
			"copywriter": {
				Name:        "copywriter",
				Description: "Vende gelo para esquimos",
				Content:     "# Regras\n1. Use gatilhos mentais",
			},
		},
	}
}

func TestSkillLoader_Success(t *testing.T) {
	tool := NewSkillLoaderTool(newMockSkillRepo())
	result, err := tool.Run("copywriter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "gatilhos mentais") {
		t.Fatalf("expected content with 'gatilhos mentais', got %q", result)
	}
}

func TestSkillLoader_NotFound(t *testing.T) {
	tool := NewSkillLoaderTool(newMockSkillRepo())
	_, err := tool.Run("nonexistent")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got %v", err)
	}
}

func TestSkillLoader_EmptyInput(t *testing.T) {
	tool := NewSkillLoaderTool(newMockSkillRepo())
	_, err := tool.Run("")
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected 'required' error, got %v", err)
	}
}

func TestSkillLoader_Metadata(t *testing.T) {
	tool := NewSkillLoaderTool(newMockSkillRepo())
	if tool.Name() != "skill_loader" {
		t.Fatalf("expected name 'skill_loader', got %q", tool.Name())
	}
	if tool.Safety() != "safe" {
		t.Fatalf("expected safety 'safe', got %q", tool.Safety())
	}
}
