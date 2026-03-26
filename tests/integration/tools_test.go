package integration

import (
	"os"
	"path/filepath"
	"testing"

	agent "github.com/renesul/ok/infrastructure/agent"
	agenttools "github.com/renesul/ok/infrastructure/agent/tools"
	"github.com/renesul/ok/infrastructure/llm"
	"go.uber.org/zap"
)

// --- SearchTool ---

func TestSearchToolBasic(t *testing.T) {
	dir := setupSearchDir(t)
	defer os.RemoveAll(dir)

	tool := agenttools.NewSearchTool("")
	input := `{"directory":"` + dir + `","pattern":"hello"}`
	result, err := tool.Run(input)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if !containsSubstring(result, "1 resultados") {
		t.Errorf("expected 1 result, got: %s", result)
	}
	if !containsSubstring(result, "test.go") {
		t.Errorf("expected test.go in results, got: %s", result)
	}
}

func TestSearchToolNoResults(t *testing.T) {
	dir := setupSearchDir(t)
	defer os.RemoveAll(dir)

	tool := agenttools.NewSearchTool("")
	input := `{"directory":"` + dir + `","pattern":"unicornio_inexistente"}`
	result, err := tool.Run(input)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if result != "no results found" {
		t.Errorf("expected 'no results found', got: %s", result)
	}
}

func TestSearchToolFileExtensionFilter(t *testing.T) {
	dir := setupSearchDir(t)
	defer os.RemoveAll(dir)

	tool := agenttools.NewSearchTool("")
	input := `{"directory":"` + dir + `","pattern":"content","file_extension":".txt"}`
	result, err := tool.Run(input)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if !containsSubstring(result, "notes.txt") {
		t.Errorf("expected notes.txt in results, got: %s", result)
	}
	if containsSubstring(result, "test.go") {
		t.Errorf("should not include .go file when filtering by .txt")
	}
}

func TestSearchToolBlockedPath(t *testing.T) {
	tool := agenttools.NewSearchTool("")
	input := `{"directory":"/etc","pattern":"root"}`
	_, err := tool.Run(input)
	if err == nil {
		t.Error("expected error for blocked path /etc")
	}
}

func TestSearchToolMissingPattern(t *testing.T) {
	tool := agenttools.NewSearchTool("")
	input := `{"directory":"/tmp","pattern":""}`
	_, err := tool.Run(input)
	if err == nil {
		t.Error("expected error for empty pattern")
	}
}

func TestSearchToolRegex(t *testing.T) {
	dir := setupSearchDir(t)
	defer os.RemoveAll(dir)

	tool := agenttools.NewSearchTool("")
	input := `{"directory":"` + dir + `","pattern":"func.*main"}`
	result, err := tool.Run(input)
	if err != nil {
		t.Fatalf("regex search failed: %v", err)
	}
	if !containsSubstring(result, "1 resultados") {
		t.Errorf("expected 1 regex result, got: %s", result)
	}
}

func setupSearchDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "search-test-*")
	if err != nil {
		t.Fatalf("create dir: %v", err)
	}
	os.WriteFile(filepath.Join(dir, "test.go"), []byte("package main\n\nfunc main() {\n\t// hello world\n}\n"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("some content here\nanother line\n"), 0644)
	return dir
}

// --- FileEditTool ---

func TestFileEditToolBasic(t *testing.T) {
	dir := setupSandbox(t)
	defer os.RemoveAll(dir)

	filePath := filepath.Join(dir, "edit_test.txt")
	os.WriteFile(filePath, []byte("line1\nline2\nline3\nline4\n"), 0644)

	tool := agenttools.NewFileEditTool(nil)
	input := `{"file":"` + filePath + `","start_line":2,"end_line":3,"replacement":"new_line2\nnew_line3"}`
	result, err := tool.Run(input)
	if err != nil {
		t.Fatalf("file edit failed: %v", err)
	}
	if !containsSubstring(result, "editado") {
		t.Errorf("expected 'editado' in result, got: %s", result)
	}

	data, _ := os.ReadFile(filePath)
	content := string(data)
	if !containsSubstring(content, "new_line2") {
		t.Errorf("expected edited content, got: %s", content)
	}
	if !containsSubstring(content, "line1") {
		t.Errorf("line1 should be preserved, got: %s", content)
	}
	if !containsSubstring(content, "line4") {
		t.Errorf("line4 should be preserved, got: %s", content)
	}
}

func TestFileEditToolBlockedPath(t *testing.T) {
	tool := agenttools.NewFileEditTool(nil)
	input := `{"file":"/etc/passwd","start_line":1,"end_line":1,"content":"evil"}`
	_, err := tool.Run(input)
	if err == nil {
		t.Error("expected error for blocked system path")
	}
}

func TestFileEditToolInvalidLineRange(t *testing.T) {
	tool := agenttools.NewFileEditTool(nil)
	input := `{"file":"/tmp/test.txt","start_line":0,"end_line":1,"content":"test"}`
	_, err := tool.Run(input)
	if err == nil {
		t.Error("expected error for start_line < 1")
	}
}

func TestFileEditToolEndBeforeStart(t *testing.T) {
	tool := agenttools.NewFileEditTool(nil)
	input := `{"file":"/tmp/test.txt","start_line":5,"end_line":2,"content":"test"}`
	_, err := tool.Run(input)
	if err == nil {
		t.Error("expected error for end_line < start_line")
	}
}

// --- LearnRuleTool ---

func TestLearnRuleToolBasic(t *testing.T) {
	defer cleanupMemory(t)

	memory := agent.NewSQLiteMemory(testDB, zap.NewNop())
	tool := agenttools.NewLearnRuleTool(memory)

	result, err := tool.Run(`{"rule":"sempre responder em portugues"}`)
	if err != nil {
		t.Fatalf("learn rule failed: %v", err)
	}
	if !containsSubstring(result, "regra aprendida") {
		t.Errorf("expected 'regra aprendida', got: %s", result)
	}

	// Verify rule was saved with category "rule"
	entries, err := memory.SearchByCategory("", "rule", 10)
	if err != nil {
		t.Fatalf("search by category failed: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.Content == "sempre responder em portugues" && e.Category == "rule" {
			found = true
			break
		}
	}
	if !found {
		t.Error("rule not found in memory with category 'rule'")
	}
}

func TestLearnRuleToolEmptyRule(t *testing.T) {
	memory := agent.NewSQLiteMemory(testDB, zap.NewNop())
	tool := agenttools.NewLearnRuleTool(memory)

	_, err := tool.Run(`{"rule":""}`)
	if err == nil {
		t.Error("expected error for empty rule")
	}
}

func TestLearnRuleToolInvalidJSON(t *testing.T) {
	memory := agent.NewSQLiteMemory(testDB, zap.NewNop())
	tool := agenttools.NewLearnRuleTool(memory)

	_, err := tool.Run("not json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// --- REPLTool ---

func TestREPLToolPython(t *testing.T) {
	tool := agenttools.NewREPLTool(nil)
	result, err := tool.Run(`{"language":"python","code":"print(2+2)"}`)
	if err != nil {
		t.Skipf("skipping REPL test (python3 not available): %v", err)
	}
	if !containsSubstring(result, "4") {
		t.Errorf("expected '4' in output, got: %s", result)
	}
}

func TestREPLToolBash(t *testing.T) {
	tool := agenttools.NewREPLTool(nil)
	result, err := tool.Run(`{"language":"bash","code":"echo hello"}`)
	if err != nil {
		t.Skipf("skipping REPL test (bash not available): %v", err)
	}
	if !containsSubstring(result, "hello") {
		t.Errorf("expected 'hello' in output, got: %s", result)
	}
}

func TestREPLToolInvalidLanguage(t *testing.T) {
	tool := agenttools.NewREPLTool(nil)
	_, err := tool.Run(`{"language":"ruby","code":"puts 1"}`)
	if err == nil {
		t.Error("expected error for unsupported language")
	}
}

func TestREPLToolEmptyCode(t *testing.T) {
	tool := agenttools.NewREPLTool(nil)
	_, err := tool.Run(`{"language":"python","code":""}`)
	if err == nil {
		t.Error("expected error for empty code")
	}
}

// --- BrowserTool ---

func TestBrowserToolEmptyURL(t *testing.T) {
	tool := agenttools.NewBrowserTool(nil, llm.ClientConfig{})
	_, err := tool.Run(`{"url":""}`)
	if err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestBrowserToolBlocksLocalhost(t *testing.T) {
	tool := agenttools.NewBrowserTool(nil, llm.ClientConfig{})
	_, err := tool.Run(`{"url":"http://localhost:8080"}`)
	if err == nil {
		t.Error("expected error for localhost URL")
	}
}

func TestBrowserToolBlocksPrivateIP(t *testing.T) {
	tool := agenttools.NewBrowserTool(nil, llm.ClientConfig{})
	_, err := tool.Run(`{"url":"http://192.168.1.1"}`)
	if err == nil {
		t.Error("expected error for private IP")
	}
}

func TestBrowserToolInvalidProtocol(t *testing.T) {
	tool := agenttools.NewBrowserTool(nil, llm.ClientConfig{})
	_, err := tool.Run(`{"url":"ftp://example.com"}`)
	if err == nil {
		t.Error("expected error for non-http protocol")
	}
}
