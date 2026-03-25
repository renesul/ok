package tools

import (
	"strings"
	"testing"
)

func TestShellTool_Tier1Blocked(t *testing.T) {
	tool := NewShellTool()
	blocked := []string{
		"rm -rf /",
		"rm -rf /*",
		"dd if=/dev/zero of=/dev/sda",
		"mkfs.ext4 /dev/sda",
		"> /etc/passwd",
		"> /proc/sys/kernel",
		"> /sys/class",
	}
	for _, cmd := range blocked {
		_, err := tool.Run(cmd)
		if err == nil {
			t.Errorf("expected Tier 1 block for %q", cmd)
		}
		if err != nil && !strings.Contains(err.Error(), "bloqueado") {
			t.Errorf("cmd %q: error = %q, want 'bloqueado'", cmd, err.Error())
		}
	}
}

func TestShellTool_Tier2RequiresConfirmation(t *testing.T) {
	tool := NewShellTool() // no confirmation manager
	needConfirm := []string{
		"sudo apt update",
		"rm -rf /tmp/test",
		"chmod 777 file",
		"chown root file",
		"kill -9 1234",
		"shutdown -h now",
		"reboot",
	}
	for _, cmd := range needConfirm {
		_, err := tool.Run(cmd)
		if err == nil {
			t.Errorf("expected Tier 2 confirmation error for %q", cmd)
		}
		if err != nil && !strings.Contains(err.Error(), "confirmacao") {
			t.Errorf("cmd %q: error = %q, want 'confirmacao'", cmd, err.Error())
		}
	}
}

func TestShellTool_Tier3Allowed(t *testing.T) {
	tool := NewShellTool()
	result, err := tool.Run("echo hello world")
	if err != nil {
		t.Fatalf("echo should succeed: %v", err)
	}
	if !strings.Contains(result, "hello world") {
		t.Errorf("result = %q, want to contain 'hello world'", result)
	}
}

func TestShellTool_Tier3PipesAllowed(t *testing.T) {
	tool := NewShellTool()
	result, err := tool.Run("echo 'abc' | wc -c")
	if err != nil {
		t.Fatalf("pipes should work: %v", err)
	}
	if strings.TrimSpace(result) == "" {
		t.Error("expected non-empty output from pipe")
	}
}

func TestShellTool_EmptyInput(t *testing.T) {
	tool := NewShellTool()
	_, err := tool.Run("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}
