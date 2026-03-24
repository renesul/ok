package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/renesul/ok/domain"
)

// --- GET /api/agent/status ---

func TestAgentStatusEndpoint(t *testing.T) {
	defer cleanupSessions(t)

	resp := authenticatedRequest(t, "GET", "/api/agent/status", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)

	if body["state"] != "idle" {
		t.Errorf("expected state 'idle', got '%v'", body["state"])
	}
	if _, ok := body["whatsapp_enabled"]; !ok {
		t.Error("expected whatsapp_enabled field")
	}
	if _, ok := body["telegram_enabled"]; !ok {
		t.Error("expected telegram_enabled field")
	}
	if _, ok := body["discord_enabled"]; !ok {
		t.Error("expected discord_enabled field")
	}
}

func TestAgentStatusRequiresAuth(t *testing.T) {
	resp := unauthenticatedRequest(t, "GET", "/api/agent/status")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// --- GET /api/agent/metrics ---

func TestAgentMetricsEndpoint(t *testing.T) {
	defer cleanupSessions(t)

	resp := authenticatedRequest(t, "GET", "/api/agent/metrics", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)

	if _, ok := body["total_executions"]; !ok {
		t.Error("expected total_executions field")
	}
	if _, ok := body["tool_usage_count"]; !ok {
		t.Error("expected tool_usage_count field")
	}
}

func TestAgentMetricsRequiresAuth(t *testing.T) {
	resp := unauthenticatedRequest(t, "GET", "/api/agent/metrics")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// --- GET /api/agent/limits ---

func TestAgentGetLimitsEndpoint(t *testing.T) {
	defer cleanupSessions(t)

	resp := authenticatedRequest(t, "GET", "/api/agent/limits", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var limits domain.AgentLimits
	json.NewDecoder(resp.Body).Decode(&limits)

	if limits.MaxSteps <= 0 {
		t.Errorf("expected max_steps > 0, got %d", limits.MaxSteps)
	}
	if limits.MaxAttempts <= 0 {
		t.Errorf("expected max_attempts > 0, got %d", limits.MaxAttempts)
	}
	if limits.TimeoutMs <= 0 {
		t.Errorf("expected timeout_ms > 0, got %d", limits.TimeoutMs)
	}
}

// --- PUT /api/agent/limits ---

func TestAgentSetLimitsEndpoint(t *testing.T) {
	defer cleanupSessions(t)

	body := `{"max_steps":10,"max_attempts":5,"timeout_ms":60000}`
	resp := authenticatedRequest(t, "PUT", "/api/agent/limits", bytes.NewBufferString(body))
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var limits domain.AgentLimits
	json.NewDecoder(resp.Body).Decode(&limits)

	if limits.MaxSteps != 10 {
		t.Errorf("expected max_steps 10, got %d", limits.MaxSteps)
	}
	if limits.MaxAttempts != 5 {
		t.Errorf("expected max_attempts 5, got %d", limits.MaxAttempts)
	}
}

func TestAgentSetLimitsValidation(t *testing.T) {
	defer cleanupSessions(t)

	body := `{"max_steps":0,"max_attempts":5,"timeout_ms":60000}`
	resp := authenticatedRequest(t, "PUT", "/api/agent/limits", bytes.NewBufferString(body))
	if resp.StatusCode != 422 {
		t.Fatalf("expected 422 for invalid limits, got %d", resp.StatusCode)
	}
}

// --- GET /api/agent/config/:key ---

func TestAgentGetConfigEndpoint(t *testing.T) {
	defer cleanupSessions(t)

	resp := authenticatedRequest(t, "GET", "/api/agent/config/soul", nil)
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)

	if body["key"] != "soul" {
		t.Errorf("expected key 'soul', got '%v'", body["key"])
	}
	if body["value"] == nil || body["value"] == "" {
		t.Error("expected non-empty value for soul config")
	}
}

func TestAgentGetConfigNotFound(t *testing.T) {
	defer cleanupSessions(t)

	resp := authenticatedRequest(t, "GET", "/api/agent/config/nonexistent_key_xyz", nil)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for nonexistent key, got %d", resp.StatusCode)
	}
}

// --- PUT /api/agent/config/:key ---

func TestAgentSetConfigEndpoint(t *testing.T) {
	defer cleanupSessions(t)

	body := `{"value":"custom soul for test"}`
	resp := authenticatedRequest(t, "PUT", "/api/agent/config/soul", bytes.NewBufferString(body))
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["key"] != "soul" {
		t.Errorf("expected key 'soul', got '%v'", result["key"])
	}
	if result["value"] != "custom soul for test" {
		t.Errorf("expected value 'custom soul for test', got '%v'", result["value"])
	}

	// Verify persistence
	resp = authenticatedRequest(t, "GET", "/api/agent/config/soul", nil)
	var verify map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&verify)
	if verify["value"] != "custom soul for test" {
		t.Errorf("config not persisted: got '%v'", verify["value"])
	}
}

func TestAgentSetConfigEmptyValue(t *testing.T) {
	defer cleanupSessions(t)

	body := `{"value":""}`
	resp := authenticatedRequest(t, "PUT", "/api/agent/config/soul", bytes.NewBufferString(body))
	if resp.StatusCode != 422 {
		t.Fatalf("expected 422 for empty value, got %d", resp.StatusCode)
	}
}

// --- GET /api/agent/executions ---

func TestAgentListExecutionsEndpoint(t *testing.T) {
	defer cleanupSessions(t)
	defer cleanupExecutions(t)

	resp := authenticatedRequest(t, "GET", "/api/agent/executions", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var records []interface{}
	json.NewDecoder(resp.Body).Decode(&records)

	// Should return empty array, not null
	if records == nil {
		t.Error("expected empty array, got nil")
	}
}

// --- GET /api/agent/executions/:id ---

func TestAgentGetExecutionNotFound(t *testing.T) {
	defer cleanupSessions(t)

	resp := authenticatedRequest(t, "GET", "/api/agent/executions/nonexistent-id", nil)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for nonexistent execution, got %d", resp.StatusCode)
	}
}

func TestAgentGetExecutionFound(t *testing.T) {
	defer cleanupSessions(t)
	defer cleanupExecutions(t)

	// Insert execution directly
	record := &domain.ExecutionRecord{
		ID:        "test-exec-123",
		Goal:      "test goal",
		Status:    "done",
		TotalMs:   500,
		StepCount: 2,
		Steps:     []domain.StepResult{{Name: "step1", Tool: "echo", Status: "done"}},
		Timeline:  []domain.ExecutionEntry{{Phase: "observe", Content: "test"}},
		ToolsUsed: []string{"echo"},
		CreatedAt: time.Now(),
	}
	if err := testExecRepo.Save(record); err != nil {
		t.Fatalf("insert test execution: %v", err)
	}

	resp := authenticatedRequest(t, "GET", "/api/agent/executions/test-exec-123", nil)
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["id"] != "test-exec-123" {
		t.Errorf("expected id 'test-exec-123', got '%v'", result["id"])
	}
	if result["goal"] != "test goal" {
		t.Errorf("expected goal 'test goal', got '%v'", result["goal"])
	}
	if result["status"] != "done" {
		t.Errorf("expected status 'done', got '%v'", result["status"])
	}
}
