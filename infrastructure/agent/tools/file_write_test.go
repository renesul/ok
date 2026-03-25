package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileWriteTool_WritesFile(t *testing.T) {
	dir := t.TempDir()
	tool := NewFileWriteTool(dir)

	result, err := tool.Run(`{"path":"output.txt","content":"hello world"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "output.txt") {
		t.Errorf("result = %q, want to contain filename", result)
	}

	data, err := os.ReadFile(filepath.Join(dir, "output.txt"))
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("file content = %q, want 'hello world'", string(data))
	}
}

func TestFileWriteTool_PathTraversal(t *testing.T) {
	tool := NewFileWriteTool(t.TempDir())
	_, err := tool.Run(`{"path":"../../escape.txt","content":"bad"}`)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestFileWriteTool_SizeLimit(t *testing.T) {
	tool := NewFileWriteTool(t.TempDir())
	bigContent := strings.Repeat("x", 51*1024) // > 50KB
	_, err := tool.Run(`{"path":"big.txt","content":"` + bigContent + `"}`)
	if err == nil {
		t.Fatal("expected error for oversized content")
	}
}

func TestFileWriteTool_CreatesSubdirectory(t *testing.T) {
	dir := t.TempDir()
	tool := NewFileWriteTool(dir)

	_, err := tool.Run(`{"path":"sub/dir/file.txt","content":"nested"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "sub/dir/file.txt"))
	if string(data) != "nested" {
		t.Errorf("file content = %q, want 'nested'", string(data))
	}
}
