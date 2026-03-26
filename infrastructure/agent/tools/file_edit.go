package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/renesul/ok/domain"
	agent "github.com/renesul/ok/infrastructure/agent"
)

type FileEditTool struct {
	confirmManager *agent.ConfirmationManager
}

func NewFileEditTool(cm *agent.ConfirmationManager) *FileEditTool {
	return &FileEditTool{confirmManager: cm}
}

func (t *FileEditTool) Name() string                       { return "file_edit" }
func (t *FileEditTool) Description() string                { return "CRITICAL: edits specific lines in a file. ALWAYS use start_line and end_line for the exact snippet. NEVER rewrite the entire file to change a few lines. Input JSON: {\"file\":\"/path\", \"start_line\":N, \"end_line\":N, \"replacement\":\"new content\"}" }
func (t *FileEditTool) Safety() domain.ToolSafety          { return domain.ToolDangerous }

func (t *FileEditTool) Run(input string) (string, error) {
	return t.RunWithContext(context.Background(), input)
}

func (t *FileEditTool) RunWithContext(ctx context.Context, input string) (string, error) {
	var req struct {
		File        string `json:"file"`
		StartLine   int    `json:"start_line"`
		EndLine     int    `json:"end_line"`
		Content     string `json:"content"`
		Replacement string `json:"replacement"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("input must be JSON: {\"file\":\"/path\", \"start_line\":1, \"end_line\":5, \"replacement\":\"new code\"}")
	}

	// Aceitar replacement ou content (prioridade para replacement)
	if req.Replacement != "" {
		req.Content = req.Replacement
	}

	if req.File == "" {
		return "", fmt.Errorf("file required")
	}
	if req.StartLine < 1 {
		return "", fmt.Errorf("start_line must be >= 1")
	}
	if req.EndLine < req.StartLine {
		return "", fmt.Errorf("end_line must be >= start_line")
	}

	if err := validateEditPath(req.File); err != nil {
		return "", err
	}

	data, err := os.ReadFile(req.File)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	totalLines := len(lines)

	if req.StartLine > totalLines {
		return "", fmt.Errorf("start_line %d exceeds total lines (%d)", req.StartLine, totalLines)
	}
	if req.EndLine > totalLines {
		req.EndLine = totalLines
	}

	// Confirmacao HIL — mostrar metadata + conteudo da edicao
	if t.confirmManager != nil {
		preview := previewEditContent(req.Content)
		summary := fmt.Sprintf("file_edit %s lines %d-%d (%d lines affected):\n%s", req.File, req.StartLine, req.EndLine, req.EndLine-req.StartLine+1, preview)
		conf := t.confirmManager.Request("file_edit", summary)
		approved, waitErr := t.confirmManager.WaitForResponse(conf)
		if waitErr != nil || !approved {
			return "", fmt.Errorf("edit not approved")
		}
	}

	newContent := strings.Split(req.Content, "\n")

	result := make([]string, 0, len(lines)+len(newContent))
	result = append(result, lines[:req.StartLine-1]...)
	result = append(result, newContent...)
	result = append(result, lines[req.EndLine:]...)

	if err := os.WriteFile(req.File, []byte(strings.Join(result, "\n")), 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return fmt.Sprintf("edited: %s (lines %d-%d replaced, total %d lines)", req.File, req.StartLine, req.EndLine, len(result)), nil
}

func previewEditContent(s string) string {
	if len(s) <= 500 {
		return s
	}
	omitted := strconv.Itoa(len(s) - 400)
	return s[:200] + "\n... (" + omitted + " chars omitted) ...\n" + s[len(s)-200:]
}

func validateEditPath(file string) error {
	abs, err := filepath.Abs(file)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	blocked := []string{"/etc", "/boot", "/proc", "/sys", "/dev", "/var/run"}
	for _, b := range blocked {
		if strings.HasPrefix(abs, b+"/") || abs == b {
			return fmt.Errorf("edit blocked on system path: %s", b)
		}
	}
	return nil
}
