package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func mockServer(status int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
}

func chatResponse(content string) string {
	b, _ := json.Marshal(map[string]interface{}{
		"choices": []map[string]interface{}{
			{"message": map[string]string{"content": content}},
		},
	})
	return string(b)
}

func sseServer(chunks ...string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", mustJSON(map[string]interface{}{
				"choices": []map[string]interface{}{
					{"delta": map[string]string{"content": chunk}},
				},
			}))
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
}

func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func testClient() *Client {
	return NewClient(zap.NewNop())
}

func testConfig(url string) ClientConfig {
	return ClientConfig{BaseURL: url, APIKey: "test", Model: "test"}
}

// --- Decide ---

func TestDecide_ValidJSON(t *testing.T) {
	srv := mockServer(200, chatResponse(`{"tool":"echo","input":"hi","done":true}`))
	defer srv.Close()

	d, err := testClient().Decide(context.Background(), testConfig(srv.URL), "sys", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Tool != "echo" || d.Input != "hi" || !d.Done {
		t.Errorf("decision = %+v, want tool=echo input=hi done=true", d)
	}
}

func TestDecide_PlainTextFallback(t *testing.T) {
	srv := mockServer(200, chatResponse("Ola, como posso ajudar?"))
	defer srv.Close()

	d, err := testClient().Decide(context.Background(), testConfig(srv.URL), "sys", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !d.Done {
		t.Error("expected Done=true for plain text fallback")
	}
	if d.Input != "Ola, como posso ajudar?" {
		t.Errorf("input = %q, want plain text", d.Input)
	}
}

func TestDecide_EmptyChoices(t *testing.T) {
	srv := mockServer(200, `{"choices":[]}`)
	defer srv.Close()

	d, err := testClient().Decide(context.Background(), testConfig(srv.URL), "sys", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !d.Done {
		t.Error("expected Done=true for empty choices")
	}
}

func TestDecide_HTTPError(t *testing.T) {
	srv := mockServer(429, `{"error":"rate limited"}`)
	defer srv.Close()

	_, err := testClient().Decide(context.Background(), testConfig(srv.URL), "sys", "user")
	if err == nil {
		t.Fatal("expected error for 429")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error = %q, want to contain '429'", err.Error())
	}
}

// --- Reflect ---

func TestReflect_ValidJSON(t *testing.T) {
	srv := mockServer(200, chatResponse(`{"action":"done","final_answer":"resultado"}`))
	defer srv.Close()

	r, err := testClient().Reflect(context.Background(), testConfig(srv.URL), "sys", "ctx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Action != "done" || r.FinalAnswer != "resultado" {
		t.Errorf("reflect = %+v, want action=done", r)
	}
}

func TestReflect_ParseError(t *testing.T) {
	srv := mockServer(200, chatResponse("not json at all"))
	defer srv.Close()

	r, _ := testClient().Reflect(context.Background(), testConfig(srv.URL), "sys", "ctx")
	if r.Action != "continue" {
		t.Errorf("expected fallback action='continue', got %q", r.Action)
	}
}

func TestReflect_HTTPError(t *testing.T) {
	srv := mockServer(500, "internal error")
	defer srv.Close()

	r, _ := testClient().Reflect(context.Background(), testConfig(srv.URL), "sys", "ctx")
	if r.Action != "continue" {
		t.Errorf("expected fallback action='continue', got %q", r.Action)
	}
}

// --- ChatCompletionSync ---

func TestChatCompletionSync_Success(t *testing.T) {
	srv := mockServer(200, chatResponse("hello world"))
	defer srv.Close()

	result, err := testClient().ChatCompletionSync(context.Background(), testConfig(srv.URL), []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Errorf("result = %q, want 'hello world'", result)
	}
}

func TestChatCompletionSync_MalformedJSON(t *testing.T) {
	srv := mockServer(200, "not json{{{")
	defer srv.Close()

	_, err := testClient().ChatCompletionSync(context.Background(), testConfig(srv.URL), []Message{{Role: "user", Content: "hi"}})
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

// --- CreatePlan ---

func TestCreatePlan_ValidPlan(t *testing.T) {
	plan := `{"steps":[{"name":"s1","tool":"echo","input":"a"}],"reasoning":"plano"}`
	srv := mockServer(200, chatResponse(plan))
	defer srv.Close()

	p, err := testClient().CreatePlan(context.Background(), testConfig(srv.URL), "sys", "goal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(p.Steps))
	}
}

func TestCreatePlan_InvalidJSON(t *testing.T) {
	srv := mockServer(200, chatResponse("just text, no plan"))
	defer srv.Close()

	p, err := testClient().CreatePlan(context.Background(), testConfig(srv.URL), "sys", "goal")
	if err != nil {
		t.Fatalf("unexpected error (should be nil): %v", err)
	}
	if len(p.Steps) != 0 {
		t.Errorf("expected empty plan, got %d steps", len(p.Steps))
	}
}

// --- Stream ---

func TestStream_MultipleChunks(t *testing.T) {
	srv := sseServer("hello", " ", "world")
	defer srv.Close()

	var tokens []string
	cb := func(token string) error {
		tokens = append(tokens, token)
		return nil
	}

	cfg := testConfig(srv.URL)
	cfg.Temperature = 0.5
	cfg.MaxTokens = 100
	result, err := testClient().ChatCompletionStream(context.Background(), cfg, []Message{{Role: "user", Content: "hi"}}, cb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Errorf("result = %q, want 'hello world'", result)
	}
	if len(tokens) != 3 {
		t.Errorf("callback called %d times, want 3", len(tokens))
	}
}

func TestStream_EmptyChunks(t *testing.T) {
	// Server sends chunks with empty choices
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprint(w, "data: {\"choices\":[]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.Temperature = 0.5
	cfg.MaxTokens = 100
	result, err := testClient().ChatCompletionStream(context.Background(), cfg, []Message{{Role: "user", Content: "hi"}}, func(string) error { return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("result = %q, want empty for no content", result)
	}
}
