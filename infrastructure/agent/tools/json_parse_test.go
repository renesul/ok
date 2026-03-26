package tools

import (
	"strings"
	"testing"
)

func TestJSONParseTool_SimplePath(t *testing.T) {
	tool := &JSONParseTool{}
	input := `{"json": "{\"name\":\"Alice\"}", "path": "name"}`
	result, err := tool.Run(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Alice" {
		t.Errorf("result = %q, want 'Alice'", result)
	}
}

func TestJSONParseTool_NestedPath(t *testing.T) {
	tool := &JSONParseTool{}
	input := `{"json": "{\"user\":{\"email\":\"a@b.com\"}}", "path": "user.email"}`
	result, err := tool.Run(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "a@b.com" {
		t.Errorf("result = %q, want 'a@b.com'", result)
	}
}

func TestJSONParseTool_MissingPath(t *testing.T) {
	tool := &JSONParseTool{}
	input := `{"json": "{\"name\":\"Alice\"}", "path": "nonexistent"}`
	_, err := tool.Run(input)
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestJSONParseTool_InvalidJSON(t *testing.T) {
	tool := &JSONParseTool{}
	input := `{"json": "not valid json{", "path": "x"}`
	_, err := tool.Run(input)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestJSONParseTool_NoPath(t *testing.T) {
	tool := &JSONParseTool{}
	// Wrapper com json mas sem path — should pretty-print
	input := `{"json": "{\"key\": \"value\"}", "path": ""}`
	result, err := tool.Run(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "key") {
		t.Errorf("result should contain 'key', got %q", result)
	}
}

func TestJSONParseTool_Empty(t *testing.T) {
	tool := &JSONParseTool{}
	_, err := tool.Run("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}
