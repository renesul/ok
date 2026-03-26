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
func (t *FileWriteTool) Description() string  { return "Creates or overwrites a file in the sandbox. Input JSON: {\"path\":\"name.txt\", \"content\":\"content\"}. Path is relative to sandbox — DO NOT use absolute paths or ~/." }
func (t *FileWriteTool) Safety() domain.ToolSafety          { return domain.ToolRestricted }

type fileWriteInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (t *FileWriteTool) Run(input string) (string, error) {
	var req fileWriteInput
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("input must be JSON {path, content}: %w", err)
	}

	if req.Path == "" {
		return "", fmt.Errorf("empty path")
	}
	if len(req.Content) > maxWriteSize {
		return "", fmt.Errorf("content too large (max %d bytes)", maxWriteSize)
	}

	safe, err := safePath(t.baseDir, req.Path)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(safe)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(safe, []byte(req.Content), 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return "file written: " + req.Path, nil
}
