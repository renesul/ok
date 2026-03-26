package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupSearchDir(t *testing.T) string {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello world\ngoodbye"), 0644)
	os.WriteFile(filepath.Join(dir, "code.go"), []byte("func main() {\n\tfmt.Println(\"hello\")\n}"), 0644)
	os.WriteFile(filepath.Join(dir, "empty.txt"), []byte(""), 0644)
	return dir
}

func TestSearch_FindsPattern(t *testing.T) {
	dir := setupSearchDir(t)
	tool := NewSearchTool(dir)
	result, err := tool.Run(`{"directory":"` + dir + `","pattern":"hello"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "hello") {
		t.Fatalf("expected result to contain 'hello', got %q", result)
	}
}

func TestSearch_RegexPattern(t *testing.T) {
	dir := setupSearchDir(t)
	tool := NewSearchTool(dir)
	result, err := tool.Run(`{"directory":"` + dir + `","pattern":"hel+o"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "hello") {
		t.Fatalf("expected regex match for 'hel+o', got %q", result)
	}
}

func TestSearch_NoResults(t *testing.T) {
	dir := setupSearchDir(t)
	tool := NewSearchTool(dir)
	result, err := tool.Run(`{"directory":"` + dir + `","pattern":"zzz_nonexistent"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "no results") {
		t.Fatalf("expected 'no results', got %q", result)
	}
}

func TestSearch_FileExtensionFilter(t *testing.T) {
	dir := setupSearchDir(t)
	tool := NewSearchTool(dir)
	result, err := tool.Run(`{"directory":"` + dir + `","pattern":"hello","file_extension":".go"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "hello.txt") {
		t.Fatalf("expected .txt file to be excluded, got %q", result)
	}
}

func TestSearch_BlockedPath(t *testing.T) {
	tool := NewSearchTool("")
	_, err := tool.Run(`{"directory":"/","pattern":"test"}`)
	if err == nil {
		t.Fatal("expected error for blocked path /")
	}
}

func TestSearch_InvalidRegex(t *testing.T) {
	dir := setupSearchDir(t)
	tool := NewSearchTool(dir)
	_, err := tool.Run(`{"directory":"` + dir + `","pattern":"[invalid"}`)
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestSearch_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	tool := NewSearchTool(dir)
	result, err := tool.Run(`{"directory":"` + dir + `","pattern":"anything"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "no results") {
		t.Fatalf("expected 'no results' for empty dir, got %q", result)
	}
}

func TestSearch_EmptyPattern(t *testing.T) {
	dir := setupSearchDir(t)
	tool := NewSearchTool(dir)
	_, err := tool.Run(`{"directory":"` + dir + `","pattern":""}`)
	if err == nil {
		t.Fatal("expected error for empty pattern")
	}
}
