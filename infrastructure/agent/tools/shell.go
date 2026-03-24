package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sync"
	"time"

	"github.com/creack/pty/v2"
	"github.com/renesul/ok/domain"
	agent "github.com/renesul/ok/infrastructure/agent"
)

const (
	shellTimeout   = 10 * time.Second
	maxShellOutput = 2000
	ptyReadSize    = 512
)

// Tier 1: Sempre bloqueado
var alwaysBlockedPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\brm\s+(-[^\s]*[rf][^\s]*\s+)?/\s*$`),
	regexp.MustCompile(`\brm\s+(-[^\s]*[rf][^\s]*\s+)?/\*`),
	regexp.MustCompile(`\bdd\s+if=/dev/`),
	regexp.MustCompile(`\bmkfs\b`),
	regexp.MustCompile(`:\(\)\s*\{.*\|.*&`),
	regexp.MustCompile(`>\s*/dev/sd`),
	regexp.MustCompile(`>\s*/dev/nvme`),
	regexp.MustCompile(`>\s*/etc/`),
	regexp.MustCompile(`>\s*/proc/`),
	regexp.MustCompile(`>\s*/sys/`),
}

// Tier 2: Requer confirmacao
var confirmationPatterns = []struct {
	Pattern *regexp.Regexp
	Reason  string
}{
	{regexp.MustCompile(`\brm\b.*-[^\s]*[rf]`), "rm com flag destrutiva"},
	{regexp.MustCompile(`\bsudo\b`), "uso de sudo"},
	{regexp.MustCompile(`\bchmod\b`), "alteracao de permissoes"},
	{regexp.MustCompile(`\bchown\b`), "alteracao de proprietario"},
	{regexp.MustCompile(`\bkill\b`), "encerrar processo"},
	{regexp.MustCompile(`\bshutdown\b`), "shutdown"},
	{regexp.MustCompile(`\breboot\b`), "reboot"},
}

type ShellTool struct {
	confirmManager *agent.ConfirmationManager
	mu             sync.Mutex
	streamCb       domain.StreamCallback
}

func NewShellTool() *ShellTool {
	return &ShellTool{}
}

func NewShellToolWithConfirmation(cm *agent.ConfirmationManager) *ShellTool {
	return &ShellTool{confirmManager: cm}
}

func (t *ShellTool) Name() string                        { return "shell" }
func (t *ShellTool) Description() string                 { return "Executa comandos bash no sistema. ALERTA: Se o comando retornar output muito grande (>500 linhas), prefira usar a tool search ou filtrar com grep/head/tail. NAO injete dumps gigantes no contexto." }
func (t *ShellTool) Safety() domain.ToolSafety           { return domain.ToolDangerous }

func (t *ShellTool) SetStreamCallback(cb domain.StreamCallback) {
	t.mu.Lock()
	t.streamCb = cb
	t.mu.Unlock()
}

func (t *ShellTool) Run(input string) (string, error) {
	return t.RunWithContext(context.Background(), input)
}

func (t *ShellTool) RunWithContext(ctx context.Context, input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("comando vazio")
	}

	// Tier 1: Sempre bloqueado
	if reason := isAlwaysBlocked(input); reason != "" {
		return "", fmt.Errorf("comando bloqueado: %s", reason)
	}

	// Tier 2: Requer confirmacao
	if reason := requiresConfirmation(input); reason != "" {
		if t.confirmManager == nil {
			return "", fmt.Errorf("comando requer confirmacao: %s", reason)
		}
		conf := t.confirmManager.Request("shell", input)
		approved, err := t.confirmManager.WaitForResponse(conf)
		if err != nil || !approved {
			return "", fmt.Errorf("comando nao aprovado: %s", reason)
		}
	}

	execCtx, cancel := context.WithTimeout(ctx, shellTimeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "sh", "-c", input)

	// Tentar PTY para output com cores ANSI
	ptmx, err := pty.Start(cmd)
	if err != nil {
		// Fallback para CombinedOutput se PTY falhar
		output, execErr := cmd.CombinedOutput()
		if execCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("timeout: comando excedeu %s", shellTimeout)
		}
		if execErr != nil {
			return "", fmt.Errorf("comando falhou: %w (output: %s)", execErr, agent.TruncateWithEllipsis(string(output), maxShellOutput))
		}
		return agent.TruncateWithEllipsis(string(output), maxShellOutput), nil
	}
	defer ptmx.Close()

	// Ler do PTY em chunks, emitindo stream
	var buf bytes.Buffer
	chunk := make([]byte, ptyReadSize)

	for {
		n, readErr := ptmx.Read(chunk)
		if n > 0 {
			data := chunk[:n]
			buf.Write(data)

			t.mu.Lock()
			cb := t.streamCb
			t.mu.Unlock()
			if cb != nil {
				cb(string(data))
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				// Erro de leitura normal quando processo termina
			}
			break
		}
	}

	// Esperar processo terminar
	waitErr := cmd.Wait()

	if execCtx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("timeout: comando excedeu %s", shellTimeout)
	}
	if waitErr != nil {
		return "", fmt.Errorf("comando falhou: %w (output: %s)", waitErr, agent.TruncateWithEllipsis(buf.String(), maxShellOutput))
	}

	return agent.TruncateWithEllipsis(buf.String(), maxShellOutput), nil
}

func isAlwaysBlocked(input string) string {
	for _, pattern := range alwaysBlockedPatterns {
		if pattern.MatchString(input) {
			return pattern.String()
		}
	}
	return ""
}

func requiresConfirmation(input string) string {
	for _, entry := range confirmationPatterns {
		if entry.Pattern.MatchString(input) {
			return entry.Reason
		}
	}
	return ""
}
