package agent

import (
	"errors"
	"strings"
	"testing"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

type testTool struct {
	name   string
	safety domain.ToolSafety
}

func (t *testTool) Name() string                  { return t.name }
func (t *testTool) Description() string            { return "test" }
func (t *testTool) Run(input string) (string, error) { return "", nil }
func (t *testTool) Safety() domain.ToolSafety      { return t.safety }

func TestSafetyGate_SafeTool(t *testing.T) {
	gate := NewSafetyGate(zap.NewNop())
	tool := &testTool{name: "echo", safety: domain.ToolSafe}

	if err := gate.Check(tool, "hello"); err != nil {
		t.Errorf("safe tool should pass: %v", err)
	}
}

func TestSafetyGate_DangerousTool(t *testing.T) {
	gate := NewSafetyGate(zap.NewNop())
	tool := &testTool{name: "shell", safety: domain.ToolDangerous}

	err := gate.Check(tool, "rm -rf /tmp/test")
	if err == nil {
		t.Fatal("dangerous tool should require confirmation")
	}
	if !errors.Is(err, ErrRequiresConfirmation) {
		t.Errorf("error should be ErrRequiresConfirmation, got %v", err)
	}
}

func TestSafetyGate_HTTP_BlocksLocalhost(t *testing.T) {
	gate := NewSafetyGate(zap.NewNop())
	tool := &testTool{name: "http", safety: domain.ToolRestricted}

	blocked := []string{
		"http://localhost:8080/api",
		"http://127.0.0.1:3000",
		"http://0.0.0.0",
	}
	for _, url := range blocked {
		err := gate.Check(tool, url)
		if err == nil {
			t.Errorf("expected block for %q", url)
		}
		if err != nil && !strings.Contains(err.Error(), "blocked") {
			t.Errorf("url %q: error = %q, want 'blocked'", url, err.Error())
		}
	}
}

func TestSafetyGate_HTTP_BlocksPrivateIP(t *testing.T) {
	gate := NewSafetyGate(zap.NewNop())
	tool := &testTool{name: "http", safety: domain.ToolRestricted}

	blocked := []string{
		"http://10.0.0.1/api",
		"http://192.168.1.1",
		"http://172.16.0.1",
	}
	for _, url := range blocked {
		err := gate.Check(tool, url)
		if err == nil {
			t.Errorf("expected block for private IP %q", url)
		}
	}
}

func TestSafetyGate_HTTP_AllowsPublic(t *testing.T) {
	gate := NewSafetyGate(zap.NewNop())
	tool := &testTool{name: "http", safety: domain.ToolRestricted}

	if err := gate.Check(tool, "https://api.example.com/data"); err != nil {
		t.Errorf("public URL should be allowed: %v", err)
	}
}
