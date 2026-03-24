package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateConversation(t *testing.T) {
	defer cleanupAll(t)

	body := `{"title":"Minha conversa"}`
	resp := authenticatedRequest(t, "POST", "/api/conversations", bytes.NewBufferString(body))

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	var conversation map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&conversation)

	if conversation["title"] != "Minha conversa" {
		t.Errorf("expected title 'Minha conversa', got '%v'", conversation["title"])
	}
	if conversation["source"] != "chat" {
		t.Errorf("expected source 'chat', got '%v'", conversation["source"])
	}
	if conversation["channel"] != "web" {
		t.Errorf("expected channel 'web', got '%v'", conversation["channel"])
	}
}

func TestCreateConversationAndList(t *testing.T) {
	defer cleanupAll(t)

	body := `{"title":"Conversa teste"}`
	authenticatedRequest(t, "POST", "/api/conversations", bytes.NewBufferString(body))

	resp := authenticatedRequest(t, "GET", "/api/conversations", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var conversations []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&conversations)

	if len(conversations) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(conversations))
	}
	if conversations[0]["title"] != "Conversa teste" {
		t.Errorf("expected 'Conversa teste', got '%v'", conversations[0]["title"])
	}
}

func TestCreateConversationEmptyTitle(t *testing.T) {
	defer cleanupAll(t)

	body := `{}`
	resp := authenticatedRequest(t, "POST", "/api/conversations", bytes.NewBufferString(body))

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	var conversation map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&conversation)

	if conversation["title"] != "Nova conversa" {
		t.Errorf("expected default title 'Nova conversa', got '%v'", conversation["title"])
	}
}

func TestSendMessageReturnsSSE(t *testing.T) {
	if testCfg.LLMBaseURL == "" || testCfg.LLMAPIKey == "" {
		t.Skip("LLM not configured")
	}
	defer cleanupAll(t)

	cookie := loginAndGetCookie(t)

	// Create conversation
	createBody := `{"title":"Test"}`
	req := httptest.NewRequest("POST", "/api/conversations", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "ok_session="+cookie)
	resp, _ := testApp.Test(req)

	var conversation map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&conversation)
	convID := conversation["id"].(float64)

	// Send message
	msgBody := `{"content":"diga oi"}`
	req = httptest.NewRequest("POST", "/api/conversations/"+itoa(int(convID))+"/messages", bytes.NewBufferString(msgBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "ok_session="+cookie)
	resp, _ = testApp.Test(req, -1)

	respBody, _ := io.ReadAll(resp.Body)
	bodyStr := string(respBody)

	// The SSE stream should contain data events
	if !strings.Contains(bodyStr, "data:") {
		t.Fatalf("expected SSE data events, got: %s", bodyStr)
	}
}
