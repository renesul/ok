package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/creack/pty/v2"
	"github.com/renesul/ok/domain"
	agent "github.com/renesul/ok/infrastructure/agent"
)

const (
	replTimeout   = 10 * time.Second
	maxREPLOutput = 4000
)

var langConfig = map[string]struct {
	binary string
	ext    string
}{
	"python": {"python3", ".py"},
	"node":   {"node", ".js"},
	"bash":   {"bash", ".sh"},
}

type REPLTool struct {
	confirmManager *agent.ConfirmationManager
	mu             sync.Mutex
	streamCb       domain.StreamCallback
}

func NewREPLTool(cm *agent.ConfirmationManager) *REPLTool {
	return &REPLTool{confirmManager: cm}
}

func (t *REPLTool) Name() string                       { return "repl" }
func (t *REPLTool) Description() string {
	return "executes scripts and returns the output. Input JSON: {\"language\":\"node\", \"code\":\"console.log(42)\"}. Languages: python, node (JavaScript), bash. IMPORTANT: use the exact language requested by the user."
}
func (t *REPLTool) Safety() domain.ToolSafety          { return domain.ToolDangerous }

func (t *REPLTool) SetStreamCallback(cb domain.StreamCallback) {
	t.mu.Lock()
	t.streamCb = cb
	t.mu.Unlock()
}

func (t *REPLTool) Run(input string) (string, error) {
	return t.RunWithContext(context.Background(), input)
}

func (t *REPLTool) RunWithContext(ctx context.Context, input string) (string, error) {
	var req struct {
		Language string `json:"language"`
		Script   string `json:"script"`
		Code     string `json:"code"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("input deve ser JSON: {\"language\":\"python\", \"code\":\"print(2+2)\"}")
	}

	if req.Code != "" {
		req.Script = req.Code
	}

	if req.Script == "" {
		return "", fmt.Errorf("code obrigatorio")
	}

	lang, ok := langConfig[req.Language]
	if !ok {
		return "", fmt.Errorf("language deve ser: python, node ou bash")
	}

	// Confirmacao HIL
	if t.confirmManager != nil {
		summary := fmt.Sprintf("repl %s (%d chars):\n%s", req.Language, len(req.Script), previewCode(req.Script))
		conf := t.confirmManager.Request("repl", summary)
		approved, waitErr := t.confirmManager.WaitForResponse(conf)
		if waitErr != nil || !approved {
			return "", fmt.Errorf("execucao nao aprovada")
		}
	}

	tempFile, err := os.CreateTemp("", fmt.Sprintf("ok_repl_*%s", lang.ext))
	if err != nil {
		return "", fmt.Errorf("criar script temporario seguro: %w", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write([]byte(req.Script)); err != nil {
		tempFile.Close()
		return "", fmt.Errorf("escrever script: %w", err)
	}
	tempFile.Close()

	execCtx, cancel := context.WithTimeout(ctx, replTimeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, lang.binary, tempFile.Name())

	// Tentar PTY para output com cores ANSI
	ptmx, ptyErr := pty.Start(cmd)
	if ptyErr != nil {
		// Fallback
		output, err := cmd.CombinedOutput()
		if execCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("timeout: script excedeu %s", replTimeout)
		}
		result := agent.TruncateWithEllipsis(string(output), maxREPLOutput)
		if err != nil {
			exitCode := -1
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
			return fmt.Sprintf("exit code %d: %s", exitCode, result), nil
		}
		return result, nil
	}
	defer ptmx.Close()

	// Ler do PTY em chunks com streaming
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
				// Normal quando processo termina
			}
			break
		}
	}

	waitErr := cmd.Wait()

	if execCtx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("timeout: script excedeu %s", replTimeout)
	}

	result := agent.TruncateWithEllipsis(buf.String(), maxREPLOutput)
	if waitErr != nil {
		exitCode := -1
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return fmt.Sprintf("exit code %d: %s", exitCode, result), nil
	}

	return result, nil
}

func previewCode(s string) string {
	if len(s) <= 500 {
		return s
	}
	omitted := strconv.Itoa(len(s) - 400)
	return s[:200] + "\n... (" + omitted + " chars omitidos) ...\n" + s[len(s)-200:]
}
