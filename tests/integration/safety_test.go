package integration

import (
	"errors"
	"strings"
	"testing"

	agent "github.com/renesul/ok/infrastructure/agent"
	agenttools "github.com/renesul/ok/infrastructure/agent/tools"
	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

func TestSafetyGateSafeTool(t *testing.T) {
	gate := agent.NewSafetyGate(zap.NewNop())
	tool := &agenttools.EchoTool{}

	err := gate.Check(tool, "hello")
	if err != nil {
		t.Errorf("safe tool should pass, got: %v", err)
	}
}

func TestSafetyGateRestrictedToolValidURL(t *testing.T) {
	gate := agent.NewSafetyGate(zap.NewNop())
	tool := agenttools.NewHTTPTool()

	err := gate.Check(tool, "https://example.com")
	if err != nil {
		t.Errorf("restricted tool with valid URL should pass, got: %v", err)
	}
}

func TestSafetyGateRestrictedToolBlocksLocalhost(t *testing.T) {
	gate := agent.NewSafetyGate(zap.NewNop())
	tool := agenttools.NewHTTPTool()

	err := gate.Check(tool, "http://localhost:8080/admin")
	if err == nil {
		t.Error("restricted tool with localhost should be blocked")
	}
}

func TestSafetyGateRestrictedToolBlocksPrivateIP(t *testing.T) {
	gate := agent.NewSafetyGate(zap.NewNop())
	tool := agenttools.NewHTTPTool()

	err := gate.Check(tool, "http://192.168.1.1/api")
	if err == nil {
		t.Error("restricted tool with private IP should be blocked")
	}
}

func TestSafetyGateDangerousTool(t *testing.T) {
	gate := agent.NewSafetyGate(zap.NewNop())
	tool := agenttools.NewShellTool()

	err := gate.Check(tool, "ls -la")
	if err == nil {
		t.Error("dangerous tool should require confirmation")
	}
	if !errors.Is(err, agent.ErrRequiresConfirmation) {
		t.Errorf("expected ErrRequiresConfirmation, got: %v", err)
	}
}

func TestSafetyGateGetToolSafety(t *testing.T) {
	gate := agent.NewSafetyGate(zap.NewNop())

	tests := []struct {
		tool     domain.Tool
		expected domain.ToolSafety
	}{
		{&agenttools.EchoTool{}, domain.ToolSafe},
		{&agenttools.MathTool{}, domain.ToolSafe},
		{agenttools.NewHTTPTool(), domain.ToolRestricted},
		{agenttools.NewShellTool(), domain.ToolDangerous},
	}

	for _, tt := range tests {
		safety := gate.GetToolSafety(tt.tool)
		if safety != tt.expected {
			t.Errorf("tool %s: expected %s, got %s", tt.tool.Name(), tt.expected, safety)
		}
	}
}

func TestShellAllowsPipes(t *testing.T) {
	tool := agenttools.NewShellTool()
	result, err := tool.Run("echo hello | tr a-z A-Z")
	if err != nil {
		t.Errorf("pipes should be allowed, got: %v", err)
	}
	if !strings.Contains(result, "HELLO") {
		t.Errorf("expected result to contain 'HELLO', got '%s'", result)
	}
}

func TestShellAllowsRedirects(t *testing.T) {
	tool := agenttools.NewShellTool()
	_, err := tool.Run("echo test > /tmp/ok_test_shell.txt && cat /tmp/ok_test_shell.txt && rm /tmp/ok_test_shell.txt")
	if err != nil {
		t.Errorf("redirects to /tmp should be allowed, got: %v", err)
	}
}

func TestShellBlocksRedirectToEtc(t *testing.T) {
	tool := agenttools.NewShellTool()
	_, err := tool.Run("echo test > /etc/passwd")
	if err == nil {
		t.Error("shell should block redirects to /etc/")
	}
}

func TestShellBlocksRmRfRoot(t *testing.T) {
	tool := agenttools.NewShellTool()
	_, err := tool.Run("rm -rf /")
	if err == nil {
		t.Error("rm -rf / should always be blocked")
	}
}

func TestShellBlocksDd(t *testing.T) {
	tool := agenttools.NewShellTool()
	_, err := tool.Run("dd if=/dev/zero of=/dev/sda")
	if err == nil {
		t.Error("dd to device should always be blocked")
	}
}

func TestShellBlocksMkfs(t *testing.T) {
	tool := agenttools.NewShellTool()
	_, err := tool.Run("mkfs.ext4 /dev/sda1")
	if err == nil {
		t.Error("mkfs should always be blocked")
	}
}

func TestShellRequiresConfirmationForRm(t *testing.T) {
	// Sem ConfirmationManager, deve retornar erro
	tool := agenttools.NewShellTool()
	_, err := tool.Run("rm -rf /tmp/testdir")
	if err == nil {
		t.Error("rm -rf should require confirmation")
	}
}

func TestShellAllowsSubshell(t *testing.T) {
	tool := agenttools.NewShellTool()
	result, err := tool.Run("echo $(whoami)")
	if err != nil {
		t.Errorf("subshell should be allowed, got: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result from subshell")
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := agent.NewRateLimiter()

	// shell limit is 5/min
	for i := 0; i < 5; i++ {
		if err := limiter.Allow("shell"); err != nil {
			t.Errorf("call %d should be allowed: %v", i+1, err)
		}
	}

	// 6th call should be blocked
	if err := limiter.Allow("shell"); err == nil {
		t.Error("6th shell call should be rate limited")
	}

	// Unlimited tools should always pass
	for i := 0; i < 100; i++ {
		if err := limiter.Allow("echo"); err != nil {
			t.Errorf("echo should be unlimited: %v", err)
		}
	}
}
