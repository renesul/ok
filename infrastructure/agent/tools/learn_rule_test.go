package tools

import (
	"strings"
	"testing"
)

func TestLearnRule_EmptyRule(t *testing.T) {
	tool := NewLearnRuleTool(nil)
	_, err := tool.Run(`{"rule":""}`)
	if err == nil || !strings.Contains(err.Error(), "obrigatorio") {
		t.Fatalf("expected 'obrigatorio' error, got %v", err)
	}
}

func TestLearnRule_InvalidJSON(t *testing.T) {
	tool := NewLearnRuleTool(nil)
	_, err := tool.Run("not json")
	if err == nil || !strings.Contains(err.Error(), "JSON") {
		t.Fatalf("expected JSON error, got %v", err)
	}
}

func TestLearnRule_EmptyInput(t *testing.T) {
	tool := NewLearnRuleTool(nil)
	_, err := tool.Run("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestLearnRule_Metadata(t *testing.T) {
	tool := NewLearnRuleTool(nil)
	if tool.Name() != "learn_rule" {
		t.Fatalf("expected name 'learn_rule', got %q", tool.Name())
	}
	if tool.Safety() != "safe" {
		t.Fatalf("expected safety 'safe', got %q", tool.Safety())
	}
}
