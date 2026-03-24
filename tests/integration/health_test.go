package integration

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
)

func TestHealthServicesEndpoint(t *testing.T) {
	defer cleanupAll(t)

	cookie := loginAndGetCookie(t)
	req := httptest.NewRequest("GET", "/api/health/services", nil)
	req.Header.Set("Cookie", "ok_session="+cookie)
	resp, err := testApp.Test(req, -1) // no timeout - pings real services
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// LLM should have a status field
	llmResult, ok := result["llm"]
	if !ok {
		t.Fatal("expected 'llm' in response")
	}
	if llmResult["status"] == nil {
		t.Error("expected 'status' field in llm result")
	}

	// Embedding should have a status field
	embedResult, ok := result["embedding"]
	if !ok {
		t.Fatal("expected 'embedding' in response")
	}
	if embedResult["status"] == nil {
		t.Error("expected 'status' field in embedding result")
	}
}

func TestHealthServicesRequiresAuth(t *testing.T) {
	resp := unauthenticatedRequest(t, "GET", "/api/health/services")

	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
