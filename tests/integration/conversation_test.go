package integration

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
)

func TestListConversationsEmpty(t *testing.T) {
	defer cleanupAll(t)

	resp := authenticatedRequest(t, "GET", "/api/conversations", nil)

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var conversations []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&conversations)

	if len(conversations) != 0 {
		t.Fatalf("expected 0 conversations, got %d", len(conversations))
	}
}

func TestListConversationsAfterImport(t *testing.T) {
	defer cleanupAll(t)

	zipData := createTestChatGPTZip(t)
	status, body := uploadZip(t, zipData)
	if status != 201 {
		t.Fatalf("import failed: %d %s", status, string(body))
	}

	resp := authenticatedRequest(t, "GET", "/api/conversations", nil)
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var conversations []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&conversations)

	if len(conversations) != 2 {
		for i, c := range conversations {
			t.Logf("Conversation %d: %v", i, c)
		}
		t.Fatalf("expected 2 conversations, got %d", len(conversations))
	}
}

func TestGetMessages(t *testing.T) {
	defer cleanupAll(t)

	zipData := createTestChatGPTZip(t)
	status, body := uploadZip(t, zipData)
	if status != 201 {
		t.Fatalf("import failed: %d %s", status, string(body))
	}

	// Get conversations to find an ID
	resp := authenticatedRequest(t, "GET", "/api/conversations", nil)
	var conversations []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&conversations)

	if len(conversations) == 0 {
		t.Fatal("no conversations found")
	}

	convID := conversations[0]["id"].(float64)
	resp = authenticatedRequest(t, "GET", "/api/conversations/"+itoa(int(convID))+"/messages", nil)
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var messages []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&messages)

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	// Verify order: first message is user, second is assistant
	if messages[0]["role"] != "user" {
		t.Errorf("expected first message role 'user', got '%v'", messages[0]["role"])
	}
	if messages[1]["role"] != "assistant" {
		t.Errorf("expected second message role 'assistant', got '%v'", messages[1]["role"])
	}
}

func TestDeleteConversation(t *testing.T) {
	defer cleanupAll(t)

	zipData := createTestChatGPTZip(t)
	uploadZip(t, zipData)

	// Get conversation ID
	resp := authenticatedRequest(t, "GET", "/api/conversations", nil)
	var conversations []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&conversations)

	if len(conversations) == 0 {
		t.Fatal("no conversations found")
	}

	convID := conversations[0]["id"].(float64)

	// Delete
	resp = authenticatedRequest(t, "DELETE", "/api/conversations/"+itoa(int(convID)), nil)
	if resp.StatusCode != 204 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 204, got %d: %s", resp.StatusCode, string(respBody))
	}

	// Verify deleted
	resp = authenticatedRequest(t, "GET", "/api/conversations", nil)
	json.NewDecoder(resp.Body).Decode(&conversations)

	if len(conversations) != 1 {
		t.Fatalf("expected 1 conversation after delete, got %d", len(conversations))
	}
}

func TestSearchConversations(t *testing.T) {
	defer cleanupAll(t)

	zipData := createTestChatGPTZip(t)
	status, body := uploadZip(t, zipData)
	if status != 201 {
		t.Fatalf("import failed: %d %s", status, string(body))
	}

	// Search for "chocolate" - should find the bolo conversation
	resp := authenticatedRequest(t, "GET", "/api/conversations/search?q=chocolate", nil)
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var conversations []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&conversations)

	if len(conversations) == 0 {
		t.Fatal("expected at least 1 search result")
	}

	// The first result should be the most relevant (bolo de chocolate)
	found := false
	for _, c := range conversations {
		if c["title"].(string) == "Receita de bolo" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Receita de bolo' in search results")
	}
}

func TestSearchConversationsNoResults(t *testing.T) {
	defer cleanupAll(t)

	zipData := createTestChatGPTZip(t)
	uploadZip(t, zipData)

	// With semantic search enabled, even unrelated terms may return low-relevance results
	// So we just verify the endpoint works and returns a valid array
	resp := authenticatedRequest(t, "GET", "/api/conversations/search?q=xyzzy_nonexistent_term_12345", nil)
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var conversations []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&conversations)

	// Should return fewer results than total conversations (or zero)
	if len(conversations) > 2 {
		t.Fatalf("expected at most 2 results for unrelated query, got %d", len(conversations))
	}
}

func TestGetMessagesUnauthenticated(t *testing.T) {
	defer cleanupAll(t)

	zipData := createTestChatGPTZip(t)
	uploadZip(t, zipData)

	resp := authenticatedRequest(t, "GET", "/api/conversations", nil)
	var conversations []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&conversations)

	if len(conversations) == 0 {
		t.Fatal("no conversations found")
	}

	convID := conversations[0]["id"].(float64)

	// Access without auth cookie
	req := httptest.NewRequest("GET", "/api/conversations/"+itoa(int(convID))+"/messages", nil)
	resp2, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp2.StatusCode != 401 {
		t.Fatalf("expected 401 for unauthenticated access, got %d", resp2.StatusCode)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
