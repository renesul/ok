package tools

import (
	"strings"
	"testing"
)

func TestPythonRPA_EmptyCode(t *testing.T) {
	tool := NewPythonRPATool(t.TempDir(), nil)
	_, err := tool.Run("")
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected 'empty' error, got %v", err)
	}
}

func TestPythonRPA_Metadata(t *testing.T) {
	tool := NewPythonRPATool(t.TempDir(), nil)
	if tool.Name() != "python_rpa" {
		t.Fatalf("expected name 'python_rpa', got %q", tool.Name())
	}
	if tool.Safety() != "dangerous" {
		t.Fatalf("expected safety 'dangerous', got %q", tool.Safety())
	}
}
