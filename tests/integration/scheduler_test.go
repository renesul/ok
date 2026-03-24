package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

func TestSchedulerCreateJob(t *testing.T) {
	defer cleanupJobs(t)

	body := `{"name":"test job","task_type":"fetch_url","input":"acessar https://example.com","interval_seconds":120}`
	resp := authenticatedRequest(t, "POST", "/api/scheduler/jobs", bytes.NewBufferString(body))

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	var job map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&job)

	if job["name"] != "test job" {
		t.Errorf("expected name 'test job', got '%v'", job["name"])
	}
	if job["enabled"] != true {
		t.Errorf("expected enabled true, got '%v'", job["enabled"])
	}
}

func TestSchedulerListJobs(t *testing.T) {
	defer cleanupJobs(t)

	body := `{"name":"list test","task_type":"echo","input":"hello","interval_seconds":60}`
	authenticatedRequest(t, "POST", "/api/scheduler/jobs", bytes.NewBufferString(body))

	resp := authenticatedRequest(t, "GET", "/api/scheduler/jobs", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var jobs []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&jobs)

	if len(jobs) < 1 {
		t.Fatal("expected at least 1 job")
	}
}

func TestSchedulerDeleteJob(t *testing.T) {
	defer cleanupJobs(t)

	body := `{"name":"delete test","task_type":"echo","input":"hello","interval_seconds":60}`
	resp := authenticatedRequest(t, "POST", "/api/scheduler/jobs", bytes.NewBufferString(body))

	var job map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&job)
	id := job["id"].(string)

	resp = authenticatedRequest(t, "DELETE", "/api/scheduler/jobs/"+id, nil)
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestSchedulerCreateJobValidation(t *testing.T) {
	defer cleanupJobs(t)

	// Interval too small
	body := `{"name":"bad job","task_type":"echo","input":"hello","interval_seconds":10}`
	resp := authenticatedRequest(t, "POST", "/api/scheduler/jobs", bytes.NewBufferString(body))
	if resp.StatusCode != 422 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestSchedulerEndpointAuth(t *testing.T) {
	resp := unauthenticatedRequest(t, "GET", "/api/scheduler/jobs")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
