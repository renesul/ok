package tools

import "testing"

func TestEchoTool_ReturnsInput(t *testing.T) {
	tool := &EchoTool{}
	result, err := tool.Run("hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Fatalf("expected 'hello world', got %q", result)
	}
}

func TestEchoTool_EmptyInput(t *testing.T) {
	tool := &EchoTool{}
	result, err := tool.Run("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestEchoTool_Unicode(t *testing.T) {
	tool := &EchoTool{}
	input := "日本語 émoji 🎉"
	result, err := tool.Run(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Fatalf("expected %q, got %q", input, result)
	}
}
