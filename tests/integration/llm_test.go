package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

// authenticatedRequestLong sends request with no timeout (for LLM calls)
func authenticatedRequestLong(t *testing.T, method, path string, body io.Reader) (int, []byte) {
	t.Helper()
	cookie := loginAndGetCookie(t)
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Cookie", "ok_session="+cookie)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := testApp.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody
}

// --- Agent Direct Mode com LLM real ---

func TestAgentRunDirectMode(t *testing.T) {
	if testCfg.LLMBaseURL == "" || testCfg.LLMAPIKey == "" {
		t.Skip("LLM not configured")
	}
	defer cleanupAll(t)
	defer cleanupMemory(t)

	status, body := authenticatedRequestLong(t, "POST", "/api/agent/run", bytes.NewBufferString(`{"input":"ola, tudo bem?"}`))

	if status != 200 {
		t.Fatalf("expected 200, got %d: %s", status, string(body))
	}

	var agentResp map[string]interface{}
	json.Unmarshal(body, &agentResp)

	messages, ok := agentResp["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		t.Fatalf("expected non-empty messages, got: %+v", agentResp)
	}

	msg := messages[0].(string)
	if len(msg) < 5 {
		t.Errorf("expected meaningful response, got: '%s'", msg)
	}

	done, _ := agentResp["done"].(bool)
	if !done {
		t.Error("expected done=true for direct mode")
	}
}

// --- Agent Stream com LLM real ---

func TestAgentStreamDirectMode(t *testing.T) {
	if testCfg.LLMBaseURL == "" || testCfg.LLMAPIKey == "" {
		t.Skip("LLM not configured")
	}
	defer cleanupAll(t)
	defer cleanupMemory(t)

	_, body := authenticatedRequestLong(t, "POST", "/api/agent/stream", bytes.NewBufferString(`{"input":"quanto e 2+2? responda so o numero"}`))
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "data:") {
		t.Fatalf("expected SSE data events, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "done") {
		t.Error("expected done event in stream")
	}
}

// --- Chat SendMessage com LLM real ---

func TestChatSendMessageWithLLM(t *testing.T) {
	if testCfg.LLMBaseURL == "" || testCfg.LLMAPIKey == "" {
		t.Skip("LLM not configured")
	}
	defer cleanupAll(t)
	defer cleanupMemory(t)

	cookie := loginAndGetCookie(t)

	// Create conversation
	req := httptest.NewRequest("POST", "/api/conversations", bytes.NewBufferString(`{"title":"LLM Test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "ok_session="+cookie)
	resp, _ := testApp.Test(req)

	var conversation map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&conversation)
	convID := conversation["id"].(float64)

	// Send message (needs long timeout)
	req = httptest.NewRequest("POST", "/api/conversations/"+itoa(int(convID))+"/messages", bytes.NewBufferString(`{"content":"diga apenas: pong"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "ok_session="+cookie)
	resp, _ = testApp.Test(req, -1)

	respBody, _ := io.ReadAll(resp.Body)
	bodyStr := string(respBody)

	if strings.Contains(bodyStr, "\"type\":\"error\"") {
		t.Fatalf("expected successful response, got error: %s", bodyStr)
	}
	if len(bodyStr) < 10 {
		t.Fatalf("expected non-trivial response, got: %s", bodyStr)
	}

	// Verify messages were persisted
	req = httptest.NewRequest("GET", "/api/conversations/"+itoa(int(convID))+"/messages", nil)
	req.Header.Set("Cookie", "ok_session="+cookie)
	resp, _ = testApp.Test(req)

	var messages []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&messages)

	if len(messages) < 2 {
		t.Fatalf("expected at least 2 messages (user + assistant), got %d", len(messages))
	}

	if messages[0]["role"] != "user" {
		t.Errorf("expected first message role 'user', got '%v'", messages[0]["role"])
	}
	if messages[1]["role"] != "assistant" {
		t.Errorf("expected second message role 'assistant', got '%v'", messages[1]["role"])
	}
	if messages[1]["content"] == "" {
		t.Error("expected non-empty assistant response")
	}
}

// --- Chat Conversation Title Auto-Update ---

func TestChatConversationTitleUpdate(t *testing.T) {
	if testCfg.LLMBaseURL == "" || testCfg.LLMAPIKey == "" {
		t.Skip("LLM not configured")
	}
	defer cleanupAll(t)
	defer cleanupMemory(t)

	cookie := loginAndGetCookie(t)

	// Create conversation with empty title
	req := httptest.NewRequest("POST", "/api/conversations", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "ok_session="+cookie)
	resp, _ := testApp.Test(req)

	var conversation map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&conversation)
	convID := conversation["id"].(float64)

	// Send first message (updates title)
	req = httptest.NewRequest("POST", "/api/conversations/"+itoa(int(convID))+"/messages", bytes.NewBufferString(`{"content":"receita de bolo de chocolate"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "ok_session="+cookie)
	testApp.Test(req, -1)

	// Check title was updated
	req = httptest.NewRequest("GET", "/api/conversations", nil)
	req.Header.Set("Cookie", "ok_session="+cookie)
	resp, _ = testApp.Test(req)

	var conversations []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&conversations)

	found := false
	for _, c := range conversations {
		if c["id"].(float64) == convID {
			title := c["title"].(string)
			if title != "New conversation" && len(title) > 0 {
				found = true
			}
			break
		}
	}

	if !found {
		t.Error("expected conversation title to be updated from first message")
	}
}

// --- Agent Math via Task Mode ---

func TestAgentRunWithMathInput(t *testing.T) {
	if testCfg.LLMBaseURL == "" || testCfg.LLMAPIKey == "" {
		t.Skip("LLM not configured")
	}
	defer cleanupAll(t)
	defer cleanupMemory(t)

	status, body := authenticatedRequestLong(t, "POST", "/api/agent/run", bytes.NewBufferString(`{"input":"2+3*4"}`))

	if status != 200 {
		t.Fatalf("expected 200, got %d: %s", status, string(body))
	}

	var agentResp map[string]interface{}
	json.Unmarshal(body, &agentResp)

	done, _ := agentResp["done"].(bool)
	if !done {
		t.Error("expected done=true")
	}

	messages, ok := agentResp["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		t.Fatalf("expected non-empty messages, got: %+v", agentResp)
	}

	msg := messages[0].(string)
	if !containsSubstring(msg, "14") {
		t.Errorf("expected '14' in response, got: '%s'", msg)
	}
}
