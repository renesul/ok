package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agenttools "github.com/renesul/ok/infrastructure/agent/tools"
)

func TestIndexFolderBasic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n")
	writeFile(t, filepath.Join(dir, "util.go"), "package main\n\nfunc helper() string { return \"ok\" }\n")
	writeFile(t, filepath.Join(dir, "README.md"), "# Test Project\n")

	tool := agenttools.NewIndexFolderTool("")
	input, _ := json.Marshal(map[string]interface{}{"path": dir})
	result, err := tool.Run(string(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "main.go") {
		t.Error("expected output to contain main.go")
	}
	if !strings.Contains(result, "util.go") {
		t.Error("expected output to contain util.go")
	}
	if !strings.Contains(result, "README.md") {
		t.Error("expected output to contain README.md")
	}
	if !strings.Contains(result, "package main") {
		t.Error("expected output to contain file content")
	}
	if !strings.Contains(result, "Files: 3") {
		t.Error("expected output to show 3 files")
	}
}

func TestIndexFolderDepthLimit(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "a", "b", "c", "d", "e", "f")
	os.MkdirAll(deep, 0755)
	writeFile(t, filepath.Join(dir, "top.go"), "package top\n")
	writeFile(t, filepath.Join(dir, "a", "level1.go"), "package a\n")
	writeFile(t, filepath.Join(dir, "a", "b", "level2.go"), "package b\n")
	writeFile(t, filepath.Join(dir, "a", "b", "c", "level3.go"), "package c\n")

	tool := agenttools.NewIndexFolderTool("")
	input, _ := json.Marshal(map[string]interface{}{"path": dir, "max_depth": 2})
	result, err := tool.Run(string(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "top.go") {
		t.Error("expected top.go at depth 0")
	}
	if !strings.Contains(result, "level1.go") {
		t.Error("expected level1.go at depth 1")
	}
	if !strings.Contains(result, "level2.go") {
		t.Error("expected level2.go at depth 2")
	}
	if strings.Contains(result, "level3.go") {
		t.Error("level3.go should NOT appear (depth 3 > max_depth 2)")
	}
}

func TestIndexFolderPattern(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.go"), "package main\n")
	writeFile(t, filepath.Join(dir, "notes.txt"), "some notes\n")
	writeFile(t, filepath.Join(dir, "data.json"), "{}\n")

	tool := agenttools.NewIndexFolderTool("")
	input, _ := json.Marshal(map[string]interface{}{"path": dir, "pattern": "*.go"})
	result, err := tool.Run(string(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "main.go") {
		t.Error("expected main.go to match *.go pattern")
	}
	if strings.Contains(result, "notes.txt") {
		t.Error("notes.txt should NOT match *.go pattern")
	}
	if strings.Contains(result, "data.json") {
		t.Error("data.json should NOT match *.go pattern")
	}
}

func TestIndexFolderBlockedPath(t *testing.T) {
	tool := agenttools.NewIndexFolderTool("")
	input, _ := json.Marshal(map[string]interface{}{"path": "/etc"})
	_, err := tool.Run(string(input))
	if err == nil {
		t.Error("expected error for blocked path /etc")
	}
	if !strings.Contains(err.Error(), "bloqueado") {
		t.Errorf("expected 'bloqueado' in error, got: %s", err.Error())
	}
}

func TestIndexFolderNotExists(t *testing.T) {
	tool := agenttools.NewIndexFolderTool("")
	input, _ := json.Marshal(map[string]interface{}{"path": "/nonexistent/path/xyz"})
	_, err := tool.Run(string(input))
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
