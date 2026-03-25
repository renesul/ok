package tools

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestTimestamp_Now(t *testing.T) {
	tool := &TimestampTool{}
	for _, input := range []string{"", "now"} {
		result, err := tool.Run(input)
		if err != nil {
			t.Fatalf("input %q: unexpected error: %v", input, err)
		}
		if _, parseErr := time.Parse(time.RFC3339, result); parseErr != nil {
			t.Fatalf("input %q: result %q is not valid RFC3339", input, result)
		}
	}
}

func TestTimestamp_Unix(t *testing.T) {
	tool := &TimestampTool{}
	result, err := tool.Run("unix")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ts, parseErr := strconv.ParseInt(result, 10, 64)
	if parseErr != nil {
		t.Fatalf("result %q is not numeric", result)
	}
	if ts < 1700000000 {
		t.Fatalf("unix timestamp %d seems too old", ts)
	}
}

func TestTimestamp_UnixConvert(t *testing.T) {
	tool := &TimestampTool{}
	result, err := tool.Run("unix:1710000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "2024") {
		t.Fatalf("expected result to contain '2024', got %q", result)
	}
}

func TestTimestamp_UnixInvalid(t *testing.T) {
	tool := &TimestampTool{}
	_, err := tool.Run("unix:abc")
	if err == nil {
		t.Fatal("expected error for invalid unix timestamp")
	}
}

func TestTimestamp_ParseDate(t *testing.T) {
	tool := &TimestampTool{}
	result, err := tool.Run("parse:2024-01-15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "2024-01-15") {
		t.Fatalf("expected result to contain '2024-01-15', got %q", result)
	}
	if !strings.Contains(result, "unix:") {
		t.Fatalf("expected result to contain 'unix:', got %q", result)
	}
}

func TestTimestamp_ParseRFC3339(t *testing.T) {
	tool := &TimestampTool{}
	result, err := tool.Run("parse:2024-01-15T10:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "2024-01-15") {
		t.Fatalf("expected result to contain '2024-01-15', got %q", result)
	}
}

func TestTimestamp_ParseInvalid(t *testing.T) {
	tool := &TimestampTool{}
	_, err := tool.Run("parse:not-a-date")
	if err == nil || !strings.Contains(err.Error(), "formato") {
		t.Fatalf("expected 'formato' error, got %v", err)
	}
}
