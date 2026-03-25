package tools

import (
	"context"
	"strings"
	"testing"
	"time"
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

func TestShellTool_Tier1_ForkBomb(t *testing.T) {
	tool := NewShellTool()
	_, err := tool.Run(":(){ :|:& };:")
	if err == nil || !strings.Contains(err.Error(), "bloqueado") {
		t.Fatalf("expected fork bomb blocked, got %v", err)
	}
}

func TestShellTool_Tier1_DevNvmeWrite(t *testing.T) {
	tool := NewShellTool()
	_, err := tool.Run("> /dev/nvme0n1")
	if err == nil || !strings.Contains(err.Error(), "bloqueado") {
		t.Fatalf("expected /dev/nvme write blocked, got %v", err)
	}
}

func TestShellTool_Tier1_DevSdaWrite(t *testing.T) {
	tool := NewShellTool()
	_, err := tool.Run("> /dev/sda1")
	if err == nil || !strings.Contains(err.Error(), "bloqueado") {
		t.Fatalf("expected /dev/sda1 write blocked, got %v", err)
	}
}

func TestShellTool_Tier2_ChainedSudo(t *testing.T) {
	tool := NewShellTool()
	_, err := tool.Run("echo hello && sudo rm /tmp/x")
	if err == nil || !strings.Contains(err.Error(), "confirmacao") {
		t.Fatalf("expected chained sudo to require confirmation, got %v", err)
	}
}

func TestShellTool_Tier2_PipeSudo(t *testing.T) {
	tool := NewShellTool()
	_, err := tool.Run("echo pass | sudo -S apt install")
	if err == nil || !strings.Contains(err.Error(), "confirmacao") {
		t.Fatalf("expected piped sudo to require confirmation, got %v", err)
	}
}

func TestShellTool_Tier3_SubshellAllowed(t *testing.T) {
	tool := NewShellTool()
	result, err := tool.Run("echo $(echo subshell)")
	if err != nil {
		t.Fatalf("subshell should succeed: %v", err)
	}
	if !strings.Contains(result, "subshell") {
		t.Errorf("result = %q, want 'subshell'", result)
	}
}

func TestShellTool_Tier3_RedirectAllowed(t *testing.T) {
	tool := NewShellTool()
	dir := t.TempDir()
	result, err := tool.Run("echo redirect_test > " + dir + "/out.txt && cat " + dir + "/out.txt")
	if err != nil {
		t.Fatalf("redirect should succeed: %v", err)
	}
	if !strings.Contains(result, "redirect_test") {
		t.Errorf("result = %q, want 'redirect_test'", result)
	}
}

func TestShellTool_LargeOutput_Truncated(t *testing.T) {
	tool := NewShellTool()
	result, err := tool.Run("seq 1 10000")
	if err != nil {
		t.Fatalf("seq should succeed: %v", err)
	}
	if len(result) > maxShellOutput+3 {
		t.Errorf("output too large: %d bytes (max %d)", len(result), maxShellOutput+3)
	}
}

func TestShellTool_Timeout(t *testing.T) {
	tool := NewShellTool()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := tool.RunWithContext(ctx, "sleep 10")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestShellTool_WithConfirmation_NoManager(t *testing.T) {
	// Sem ConfirmationManager, Tier 2 rejeita direto
	tool := NewShellTool()
	_, err := tool.Run("sudo echo test")
	if err == nil || !strings.Contains(err.Error(), "confirmacao") {
		t.Fatalf("expected confirmacao error without manager, got %v", err)
	}
}
