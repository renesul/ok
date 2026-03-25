package tools

import (
	"strings"
	"testing"

	"github.com/renesul/ok/infrastructure/llm"
)

func TestBrowserTool_EmptyURL(t *testing.T) {
	tool := NewBrowserTool(nil, llm.ClientConfig{})
	_, err := tool.Run(`{"url":""}`)
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestBrowserTool_InvalidJSON(t *testing.T) {
	tool := NewBrowserTool(nil, llm.ClientConfig{})
	_, err := tool.Run("not json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestBrowserTool_LocalhostBlocked(t *testing.T) {
	tool := NewBrowserTool(nil, llm.ClientConfig{})
	blocked := []string{
		`{"url":"http://localhost:8080"}`,
		`{"url":"http://127.0.0.1:3000"}`,
		`{"url":"http://0.0.0.0"}`,
	}
	for _, input := range blocked {
		_, err := tool.Run(input)
		if err == nil {
			t.Errorf("expected block for %s", input)
		}
		if err != nil && !strings.Contains(err.Error(), "bloqueada") {
			t.Errorf("error = %q, want 'bloqueada'", err.Error())
		}
	}
}

func TestBrowserTool_NoScheme(t *testing.T) {
	tool := NewBrowserTool(nil, llm.ClientConfig{})
	_, err := tool.Run(`{"url":"example.com"}`)
	if err == nil {
		t.Fatal("expected error for URL without scheme")
	}
}

func TestBrowserTool_UnknownAction(t *testing.T) {
	tool := NewBrowserTool(nil, llm.ClientConfig{})
	// This will fail at action execution if browser exists, or at "actions require Chrome" if not
	_, err := tool.Run(`{"url":"https://example.com","actions":[{"type":"teleport"}]}`)
	if err == nil {
		t.Fatal("expected error for unknown action type or missing Chrome")
	}
}
