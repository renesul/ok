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

func TestFileWriteTool_AbsolutePathBlocked(t *testing.T) {
	tool := NewFileWriteTool(t.TempDir())
	_, err := tool.Run(`{"path":"/etc/passwd","content":"x"}`)
	if err == nil {
		t.Fatal("expected error for absolute path")
	}
}

func TestFileWriteTool_DotDotVariants(t *testing.T) {
	tool := NewFileWriteTool(t.TempDir())
	cases := []string{
		`{"path":"../escape.txt","content":"x"}`,
		`{"path":"sub/../../escape.txt","content":"x"}`,
		`{"path":"sub/../../../etc/passwd","content":"x"}`,
	}
	for _, input := range cases {
		_, err := tool.Run(input)
		if err == nil {
			t.Errorf("expected error for %s", input)
		}
	}
}

func TestFileWriteTool_EmptyPath(t *testing.T) {
	tool := NewFileWriteTool(t.TempDir())
	_, err := tool.Run(`{"path":"","content":"x"}`)
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestFileWriteTool_InvalidJSON(t *testing.T) {
	tool := NewFileWriteTool(t.TempDir())
	_, err := tool.Run("not json at all")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFileWriteTool_ExactBoundarySize(t *testing.T) {
	dir := t.TempDir()
	tool := NewFileWriteTool(dir)
	content := strings.Repeat("x", 50*1024) // exactly 50KB
	input := `{"path":"boundary.txt","content":"` + content + `"}`
	_, err := tool.Run(input)
	if err != nil {
		t.Fatalf("exact boundary size should succeed: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "boundary.txt"))
	if len(data) != 50*1024 {
		t.Errorf("file size = %d, want %d", len(data), 50*1024)
	}
}

func TestFileWriteTool_OverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	tool := NewFileWriteTool(dir)

	tool.Run(`{"path":"overwrite.txt","content":"first"}`)
	tool.Run(`{"path":"overwrite.txt","content":"second"}`)

	data, _ := os.ReadFile(filepath.Join(dir, "overwrite.txt"))
	if string(data) != "second" {
		t.Errorf("file content = %q, want 'second'", string(data))
	}
}
