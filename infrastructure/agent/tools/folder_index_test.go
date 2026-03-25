package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupIndexDir(t *testing.T) string {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "data.json"), []byte(`{"key":"val"}`), 0644)
	os.MkdirAll(filepath.Join(dir, "sub", "deep"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "nested.go"), []byte("package sub"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "deep", "deep.go"), []byte("package deep"), 0644)
	return dir
}

func TestFolderIndex_BasicIndex(t *testing.T) {
	dir := setupIndexDir(t)
	tool := NewIndexFolderTool(dir)
	result, err := tool.Run(`{"path":"` + dir + `"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, name := range []string{"main.go", "readme.txt", "data.json"} {
		if !strings.Contains(result, name) {
			t.Errorf("expected result to contain %q", name)
		}
	}
}

func TestFolderIndex_GlobPattern(t *testing.T) {
	dir := setupIndexDir(t)
	tool := NewIndexFolderTool(dir)
	result, err := tool.Run(`{"path":"` + dir + `","pattern":"*.go"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "main.go") {
		t.Fatal("expected main.go in results")
	}
	if strings.Contains(result, "readme.txt") {
		t.Fatal("expected readme.txt to be excluded by *.go filter")
	}
}

func TestFolderIndex_MaxDepth(t *testing.T) {
	dir := setupIndexDir(t)
	tool := NewIndexFolderTool(dir)
	result, err := tool.Run(`{"path":"` + dir + `","max_depth":1}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "deep.go") {
		t.Fatal("expected deep.go to be excluded by max_depth=1")
	}
}

func TestFolderIndex_BlockedPath(t *testing.T) {
	tool := NewIndexFolderTool("")
	_, err := tool.Run(`{"path":"/"}`)
	if err == nil {
		t.Fatal("expected error for blocked path /")
	}
}

func TestFolderIndex_BinaryDetection(t *testing.T) {
	dir := t.TempDir()
	// Write binary content (null bytes)
	os.WriteFile(filepath.Join(dir, "binary.dat"), []byte{0x00, 0x01, 0x02, 0xFF, 0x00}, 0644)
	tool := NewIndexFolderTool(dir)
	result, err := tool.Run(`{"path":"` + dir + `"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "metadata") {
		t.Fatalf("expected binary file to show metadata only, got %q", result)
	}
}

func TestFolderIndex_LargeFile(t *testing.T) {
	dir := t.TempDir()
	// Write file > 50KB
	os.WriteFile(filepath.Join(dir, "large.txt"), []byte(strings.Repeat("x", 55*1024)), 0644)
	tool := NewIndexFolderTool(dir)
	result, err := tool.Run(`{"path":"` + dir + `"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "metadata") {
		t.Fatalf("expected large file to show metadata only, got %q", result)
	}
}

func TestFolderIndex_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	tool := NewIndexFolderTool(dir)
	result, err := tool.Run(`{"path":"` + dir + `"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return without error, possibly empty or header-only
	_ = result
}
