package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPTool_GET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello from server")
	}))
	defer srv.Close()

	tool := NewHTTPTool()
	result, err := tool.Run(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "hello from server") {
		t.Errorf("result = %q, want to contain server response", result)
	}
}

func TestHTTPTool_POST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		fmt.Fprint(w, "created")
	}))
	defer srv.Close()

	input, _ := json.Marshal(map[string]interface{}{
		"method": "POST",
		"url":    srv.URL,
		"body":   `{"key":"value"}`,
	})
	tool := NewHTTPTool()
	result, err := tool.Run(string(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "created") {
		t.Errorf("result = %q, want 'created'", result)
	}
}

func TestHTTPTool_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(15 * time.Second)
	}))
	defer srv.Close()

	tool := &HTTPTool{client: &http.Client{Timeout: 100 * time.Millisecond}}
	_, err := tool.Run(srv.URL)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
