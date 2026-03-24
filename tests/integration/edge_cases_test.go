package integration

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

// --- Import Edge Cases ---

func TestImportChatGPTEmptyConversations(t *testing.T) {
	defer cleanupAll(t)

	data := createTestZipWithContent(t, `[]`)
	status, body := uploadZipBytes(t, data)

	if status != 201 {
		t.Fatalf("expected 201, got %d: %s", status, string(body))
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	count := result["count"].(float64)
	if count != 0 {
		t.Errorf("expected 0 imported conversations, got %v", count)
	}
}

// --- Scheduler Edge Cases ---

func TestSchedulerUpdateJob(t *testing.T) {
	defer cleanupJobs(t)
	defer cleanupSessions(t)

	body := `{"name":"update test","task_type":"echo","input":"hello","interval_seconds":120}`
	resp := authenticatedRequest(t, "POST", "/api/scheduler/jobs", bytes.NewBufferString(body))

	var job map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&job)
	id := job["id"].(string)

	updateBody := `{"interval_seconds":300}`
	resp = authenticatedRequest(t, "PUT", "/api/scheduler/jobs/"+id, bytes.NewBufferString(updateBody))
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var updated map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&updated)

	if updated["interval_seconds"].(float64) != 300 {
		t.Errorf("expected interval 300, got %v", updated["interval_seconds"])
	}
}

func TestSchedulerUpdateJobDisable(t *testing.T) {
	defer cleanupJobs(t)
	defer cleanupSessions(t)

	body := `{"name":"disable test","task_type":"echo","input":"hello","interval_seconds":120}`
	resp := authenticatedRequest(t, "POST", "/api/scheduler/jobs", bytes.NewBufferString(body))

	var job map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&job)
	id := job["id"].(string)

	updateBody := `{"enabled":false}`
	resp = authenticatedRequest(t, "PUT", "/api/scheduler/jobs/"+id, bytes.NewBufferString(updateBody))
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var updated map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&updated)

	if updated["enabled"].(bool) != false {
		t.Errorf("expected enabled=false, got %v", updated["enabled"])
	}
}

// --- Conversation Delete Cascade ---

func TestDeleteConversationCascade(t *testing.T) {
	defer cleanupAll(t)

	data := createTestChatGPTZip(t)
	uploadZip(t, data)

	resp := authenticatedRequest(t, "GET", "/api/conversations", nil)
	var conversations []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&conversations)

	if len(conversations) == 0 {
		t.Fatal("expected conversations after import")
	}

	convID := int(conversations[0]["id"].(float64))

	resp = authenticatedRequest(t, "DELETE", "/api/conversations/"+itoa(convID), nil)
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	resp = authenticatedRequest(t, "GET", "/api/conversations/"+itoa(convID)+"/messages", nil)
	var messages []interface{}
	json.NewDecoder(resp.Body).Decode(&messages)

	if len(messages) != 0 {
		t.Errorf("expected 0 messages after cascade delete, got %d", len(messages))
	}
}

// --- Health Endpoint ---

func TestHealthBasicEndpoint(t *testing.T) {
	resp := unauthenticatedRequest(t, "GET", "/health")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%v'", body["status"])
	}
}

// --- Config Endpoint ---

func TestConfigEndpoint(t *testing.T) {
	defer cleanupSessions(t)

	resp := authenticatedRequest(t, "GET", "/api/config", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)

	if _, ok := body["llm_model"]; !ok {
		t.Error("expected llm_model field")
	}
	if _, ok := body["debug"]; !ok {
		t.Error("expected debug field")
	}
}

func TestConfigEndpointRequiresAuth(t *testing.T) {
	resp := unauthenticatedRequest(t, "GET", "/api/config")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// --- Helpers ---

func createTestZipWithContent(t *testing.T, jsonContent string) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	writer, err := zipWriter.Create("conversations.json")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	writer.Write([]byte(jsonContent))
	zipWriter.Close()
	return &buf
}

func uploadZipBytes(t *testing.T, zipData *bytes.Buffer) (int, []byte) {
	t.Helper()
	cookie := loginAndGetCookie(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "chatgpt-export.zip")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	part.Write(zipData.Bytes())
	writer.Close()

	req := httptest.NewRequest("POST", "/api/import/chatgpt", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Cookie", "ok_session="+cookie)

	resp, err := testApp.Test(req, -1)
	if err != nil {
		t.Fatalf("import request failed: %v", err)
	}

	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody
}
