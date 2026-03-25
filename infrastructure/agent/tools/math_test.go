package tools

import (
	"strings"
	"testing"
)

func TestMathTool_Simple(t *testing.T) {
	tool := &MathTool{}
	result, err := tool.Run("2+3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "5" {
		t.Errorf("2+3 = %q, want 5", result)
	}
}

func TestMathTool_Parentheses(t *testing.T) {
	tool := &MathTool{}
	result, err := tool.Run("(2+3)*4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "20" {
		t.Errorf("(2+3)*4 = %q, want 20", result)
	}
}

func TestMathTool_DivisionByZero(t *testing.T) {
	tool := &MathTool{}
	_, err := tool.Run("10/0")
	if err == nil {
		t.Fatal("expected error for division by zero")
	}
	if !strings.Contains(err.Error(), "zero") {
		t.Errorf("error = %q, want to contain 'zero'", err.Error())
	}
}

func TestMathTool_Negation(t *testing.T) {
	tool := &MathTool{}
	result, err := tool.Run("-5+3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "-2" {
		t.Errorf("-5+3 = %q, want -2", result)
	}
}

func TestMathTool_Empty(t *testing.T) {
	tool := &MathTool{}
	_, err := tool.Run("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestMathTool_InvalidChars(t *testing.T) {
	tool := &MathTool{}
	_, err := tool.Run("2+abc")
	if err == nil {
		t.Fatal("expected error for invalid characters")
	}
}
