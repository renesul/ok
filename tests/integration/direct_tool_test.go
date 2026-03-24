package integration

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/renesul/ok/domain"
)

func TestEngineExecutesMathTool(t *testing.T) {
	if testCfg.LLMBaseURL == "" || testCfg.LLMAPIKey == "" {
		t.Skip("LLM not configured")
	}
	defer cleanupAll(t)
	defer cleanupMemory(t)

	status, body := authenticatedRequestLong(t, "POST", "/api/agent/run",
		bytes.NewBufferString(`{"input":"quanto e (100+50)*2"}`))

	if status != 200 {
		t.Fatalf("expected 200, got %d: %s", status, string(body))
	}

	var resp domain.AgentResponse
	json.Unmarshal(body, &resp)

	if !resp.Done {
		t.Error("expected done=true")
	}
	if len(resp.Messages) == 0 {
		t.Fatal("expected at least 1 message")
	}

	msg := resp.Messages[0]
	if !containsSubstring(msg, "300") {
		t.Errorf("expected '300' in response, got: '%s'", msg)
	}
}
