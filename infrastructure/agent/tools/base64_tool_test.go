package tools

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBase64_Encode(t *testing.T) {
	tool := &Base64Tool{}
	result, err := tool.Run(`{"action":"encode","data":"hello"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "aGVsbG8=" {
		t.Fatalf("expected 'aGVsbG8=', got %q", result)
	}
}

func TestBase64_Decode(t *testing.T) {
	tool := &Base64Tool{}
	result, err := tool.Run(`{"action":"decode","data":"aGVsbG8="}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello" {
		t.Fatalf("expected 'hello', got %q", result)
	}
}

func TestBase64_InvalidAction(t *testing.T) {
	tool := &Base64Tool{}
	_, err := tool.Run(`{"action":"xxx","data":"a"}`)
	if err == nil || !strings.Contains(err.Error(), "action") {
		t.Fatalf("expected action error, got %v", err)
	}
}

func TestBase64_EmptyInput(t *testing.T) {
	tool := &Base64Tool{}
	_, err := tool.Run("")
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected 'empty' error, got %v", err)
	}
}

func TestBase64_InvalidBase64(t *testing.T) {
	tool := &Base64Tool{}
	_, err := tool.Run(`{"action":"decode","data":"!!!invalid!!!"}`)
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestBase64_PlainStringFallback(t *testing.T) {
	tool := &Base64Tool{}
	result, err := tool.Run("raw text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := base64.StdEncoding.EncodeToString([]byte("raw text"))
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestBase64_InvalidJSON(t *testing.T) {
	tool := &Base64Tool{}
	result, err := tool.Run("{bad json")
	if err != nil {
		t.Fatalf("unexpected error on fallback: %v", err)
	}
	// Should fallback to encoding the raw string
	expected := base64.StdEncoding.EncodeToString([]byte("{bad json"))
	if result != expected {
		t.Fatalf("expected fallback encode %q, got %q", expected, result)
	}
}
