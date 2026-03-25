package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestREPLTool_BashEcho(t *testing.T) {
	tool := NewREPLTool(nil)
	input, _ := json.Marshal(map[string]string{
		"language": "bash",
		"code":     "echo hello repl",
	})

	result, err := tool.Run(string(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "hello repl") {
		t.Errorf("result = %q, want to contain 'hello repl'", result)
	}
}

func TestREPLTool_InvalidLanguage(t *testing.T) {
	tool := NewREPLTool(nil)
	input, _ := json.Marshal(map[string]string{
		"language": "cobol",
		"code":     "display hello",
	})

	_, err := tool.Run(string(input))
	if err == nil {
		t.Fatal("expected error for invalid language")
	}
}

func TestREPLTool_EmptyCode(t *testing.T) {
	tool := NewREPLTool(nil)
	input, _ := json.Marshal(map[string]string{
		"language": "bash",
		"code":     "",
	})

	_, err := tool.Run(string(input))
	if err == nil {
		t.Fatal("expected error for empty code")
	}
}
