package tools

import (
	"strings"
	"testing"
)

func TestWebSearch_EmptyQuery(t *testing.T) {
	tool := NewWebSearchTool()
	_, err := tool.Run("")
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected 'required' error, got %v", err)
	}
}

func TestWebSearch_LongQuery(t *testing.T) {
	tool := NewWebSearchTool()
	_, err := tool.Run(strings.Repeat("a", 600))
	if err == nil || !strings.Contains(err.Error(), "long") {
		t.Fatalf("expected 'long' error, got %v", err)
	}
}

func TestWebSearch_Metadata(t *testing.T) {
	tool := NewWebSearchTool()
	if tool.Name() != "web_search" {
		t.Fatalf("expected name 'web_search', got %q", tool.Name())
	}
	if tool.Safety() != "restricted" {
		t.Fatalf("expected safety 'restricted', got %q", tool.Safety())
	}
}
