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
func (t *FileEditTool) Description() string                { return "CRITICO: edita linhas especificas de um arquivo. Use SEMPRE start_line e end_line para o trecho exato. NUNCA reescreva o arquivo inteiro para alterar poucas linhas. Input JSON: {\"file\":\"/path\", \"start_line\":N, \"end_line\":N, \"replacement\":\"novo conteudo\"}" }
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
		return "", fmt.Errorf("input deve ser JSON: {\"file\":\"/path\", \"start_line\":1, \"end_line\":5, \"replacement\":\"novo codigo\"}")
	}

	// Aceitar replacement ou content (prioridade para replacement)
	if req.Replacement != "" {
		req.Content = req.Replacement
	}

	if req.File == "" {
		return "", fmt.Errorf("file obrigatorio")
	}
	if req.StartLine < 1 {
		return "", fmt.Errorf("start_line deve ser >= 1")
	}
	if req.EndLine < req.StartLine {
		return "", fmt.Errorf("end_line deve ser >= start_line")
	}

	if err := validateEditPath(req.File); err != nil {
		return "", err
	}

	data, err := os.ReadFile(req.File)
	if err != nil {
		return "", fmt.Errorf("ler arquivo: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	totalLines := len(lines)

	if req.StartLine > totalLines {
		return "", fmt.Errorf("start_line %d excede total de linhas (%d)", req.StartLine, totalLines)
	}
	if req.EndLine > totalLines {
		req.EndLine = totalLines
	}

	// Confirmacao HIL — mostrar metadata + conteudo da edicao
	if t.confirmManager != nil {
		preview := previewEditContent(req.Content)
		summary := fmt.Sprintf("file_edit %s linhas %d-%d (%d linhas afetadas):\n%s", req.File, req.StartLine, req.EndLine, req.EndLine-req.StartLine+1, preview)
		conf := t.confirmManager.Request("file_edit", summary)
		approved, waitErr := t.confirmManager.WaitForResponse(conf)
		if waitErr != nil || !approved {
			return "", fmt.Errorf("edicao nao aprovada")
		}
	}

	newContent := strings.Split(req.Content, "\n")

	result := make([]string, 0, len(lines)+len(newContent))
	result = append(result, lines[:req.StartLine-1]...)
	result = append(result, newContent...)
	result = append(result, lines[req.EndLine:]...)

	if err := os.WriteFile(req.File, []byte(strings.Join(result, "\n")), 0644); err != nil {
		return "", fmt.Errorf("escrever arquivo: %w", err)
	}

	return fmt.Sprintf("editado: %s (linhas %d-%d substituidas, total %d linhas)", req.File, req.StartLine, req.EndLine, len(result)), nil
}

func previewEditContent(s string) string {
	if len(s) <= 500 {
		return s
	}
	omitted := strconv.Itoa(len(s) - 400)
	return s[:200] + "\n... (" + omitted + " chars omitidos) ...\n" + s[len(s)-200:]
}

func validateEditPath(file string) error {
	abs, err := filepath.Abs(file)
	if err != nil {
		return fmt.Errorf("resolver path: %w", err)
	}
	blocked := []string{"/etc", "/boot", "/proc", "/sys", "/dev", "/var/run"}
	for _, b := range blocked {
		if strings.HasPrefix(abs, b+"/") || abs == b {
			return fmt.Errorf("edicao bloqueada em path do sistema: %s", b)
		}
	}
	return nil
}
