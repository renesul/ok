package tools

import (
	"encoding/json"
	"fmt"

	"github.com/renesul/ok/domain"
	"os"
	"path/filepath"
)

const maxWriteSize = 50 * 1024 // 50KB

type FileWriteTool struct {
	baseDir string
}

func NewFileWriteTool(baseDir string) *FileWriteTool {
	return &FileWriteTool{baseDir: baseDir}
}

func (t *FileWriteTool) Name() string        { return "file_write" }
func (t *FileWriteTool) Description() string  { return "Cria ou sobrescreve um arquivo no sandbox. Input JSON: {\"path\":\"nome.txt\", \"content\":\"conteudo\"}. O path e relativo ao sandbox — NAO use paths absolutos ou ~/." }
func (t *FileWriteTool) Safety() domain.ToolSafety          { return domain.ToolRestricted }

type fileWriteInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (t *FileWriteTool) Run(input string) (string, error) {
	var req fileWriteInput
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("input deve ser JSON {path, content}: %w", err)
	}

	if req.Path == "" {
		return "", fmt.Errorf("path vazio")
	}
	if len(req.Content) > maxWriteSize {
		return "", fmt.Errorf("conteudo muito grande (max %d bytes)", maxWriteSize)
	}

	safe, err := safePath(t.baseDir, req.Path)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(safe)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("criar diretorio: %w", err)
	}

	if err := os.WriteFile(safe, []byte(req.Content), 0644); err != nil {
		return "", fmt.Errorf("escrever arquivo: %w", err)
	}

	return "arquivo escrito: " + req.Path, nil
}
