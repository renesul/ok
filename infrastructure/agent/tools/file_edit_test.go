package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileEditTool_ReplaceLines(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	os.WriteFile(file, []byte("line1\nline2\nline3\nline4"), 0644)

	tool := NewFileEditTool(nil) // no confirmation manager
	input, _ := json.Marshal(map[string]interface{}{
		"file":        file,
		"start_line":  2,
		"end_line":    3,
		"replacement": "replaced2\nreplaced3",
	})

	result, err := tool.Run(string(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "edited") {
		t.Errorf("result = %q, want to contain 'edited'", result)
	}

	data, _ := os.ReadFile(file)
	content := string(data)
	if !strings.Contains(content, "replaced2") {
		t.Errorf("file should contain 'replaced2', got: %s", content)
	}
	if strings.Contains(content, "line2") {
		t.Errorf("file should NOT contain old 'line2', got: %s", content)
	}
}

func TestFileEditTool_SystemPathBlocked(t *testing.T) {
	tool := NewFileEditTool(nil)
	paths := []string{"/etc/passwd", "/proc/1/status", "/sys/class/net"}
	for _, p := range paths {
		input, _ := json.Marshal(map[string]interface{}{
			"file": p, "start_line": 1, "end_line": 1, "replacement": "x",
		})
		_, err := tool.Run(string(input))
		if err == nil {
			t.Errorf("expected block for %q", p)
		}
	}
}

func TestFileEditTool_InvalidRange(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "small.txt")
	os.WriteFile(file, []byte("one\ntwo"), 0644)

	tool := NewFileEditTool(nil)
	input, _ := json.Marshal(map[string]interface{}{
		"file": file, "start_line": 10, "end_line": 15, "replacement": "x",
	})

	_, err := tool.Run(string(input))
	if err == nil {
		t.Fatal("expected error for start_line > total lines")
	}
}
