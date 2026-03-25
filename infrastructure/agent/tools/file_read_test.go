package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileReadTool_ReadsFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("line1\nline2\nline3"), 0644)

	tool := NewFileReadTool(dir)
	result, err := tool.Run(`{"file":"test.txt"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "line1") {
		t.Errorf("result should contain 'line1', got %q", result)
	}
}

func TestFileReadTool_PathTraversal(t *testing.T) {
	tool := NewFileReadTool(t.TempDir())
	_, err := tool.Run(`{"file":"../../../etc/passwd"}`)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if !strings.Contains(err.Error(), "traversal") {
		t.Errorf("error = %q, want to contain 'traversal'", err.Error())
	}
}

func TestFileReadTool_AbsolutePath(t *testing.T) {
	tool := NewFileReadTool(t.TempDir())
	_, err := tool.Run(`{"file":"/etc/passwd"}`)
	if err == nil {
		t.Fatal("expected error for absolute path")
	}
}

func TestFileReadTool_BinaryRejected(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "img.png"), []byte("fake png"), 0644)

	tool := NewFileReadTool(dir)
	_, err := tool.Run(`{"file":"img.png"}`)
	if err == nil {
		t.Fatal("expected error for binary extension")
	}
	if !strings.Contains(err.Error(), "binario") {
		t.Errorf("error = %q, want to contain 'binario'", err.Error())
	}
}

func TestFileReadTool_Pagination(t *testing.T) {
	dir := t.TempDir()
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, "line content")
	}
	os.WriteFile(filepath.Join(dir, "big.txt"), []byte(strings.Join(lines, "\n")), 0644)

	tool := NewFileReadTool(dir)
	result, err := tool.Run(`{"file":"big.txt","start_line":5,"end_line":10}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "5\t") {
		t.Errorf("result should start at line 5, got %q", result[:50])
	}
}

func TestFileReadTool_FallbackInput(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("world"), 0644)

	tool := NewFileReadTool(dir)
	result, err := tool.Run("hello.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "world") {
		t.Errorf("result = %q, want to contain 'world'", result)
	}
}
