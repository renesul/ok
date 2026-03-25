package tools

import (
	"strings"
	"testing"

	"github.com/renesul/ok/infrastructure/llm"
)

func TestBrowserTool_EmptyURL(t *testing.T) {
	tool := NewBrowserTool(nil, llm.ClientConfig{})
	_, err := tool.Run(`{"url":""}`)
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestBrowserTool_InvalidJSON(t *testing.T) {
	tool := NewBrowserTool(nil, llm.ClientConfig{})
	_, err := tool.Run("not json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestBrowserTool_LocalhostBlocked(t *testing.T) {
	tool := NewBrowserTool(nil, llm.ClientConfig{})
	blocked := []string{
		`{"url":"http://localhost:8080"}`,
		`{"url":"http://127.0.0.1:3000"}`,
		`{"url":"http://0.0.0.0"}`,
	}
	for _, input := range blocked {
		_, err := tool.Run(input)
		if err == nil {
			t.Errorf("expected block for %s", input)
		}
		if err != nil && !strings.Contains(err.Error(), "bloqueada") {
			t.Errorf("error = %q, want 'bloqueada'", err.Error())
		}
	}
}

func TestBrowserTool_NoScheme(t *testing.T) {
	tool := NewBrowserTool(nil, llm.ClientConfig{})
	_, err := tool.Run(`{"url":"example.com"}`)
	if err == nil {
		t.Fatal("expected error for URL without scheme")
	}
}

func TestBrowserTool_ActionsWithoutChrome(t *testing.T) {
	tool := NewBrowserTool(nil, llm.ClientConfig{})
	_, err := tool.Run(`{"url":"https://example.com","actions":[{"type":"click","selector":"#btn"}]}`)
	// Se Chrome nao esta instalado, deve dar erro explicito sobre actions
	// Se Chrome esta instalado, pode falhar no click (selector nao existe) — ambos sao erros validos
	if err == nil {
		t.Fatal("expected error for action on example.com")
	}
}

func TestBrowserTool_AnalyzeWithoutVision(t *testing.T) {
	tool := NewBrowserTool(nil, llm.ClientConfig{})
	_, err := tool.Run(`{"url":"https://example.com","actions":[{"type":"analyze"}]}`)
	if err == nil {
		t.Fatal("expected error when vision not configured")
	}
}

func TestBrowserTool_JSBlocked(t *testing.T) {
	tool := NewBrowserTool(nil, llm.ClientConfig{})
	blockedScripts := []string{
		`{"url":"https://example.com","actions":[{"type":"js","script":"fetch('http://evil.com')"}]}`,
		`{"url":"https://example.com","actions":[{"type":"js","script":"document.cookie"}]}`,
		`{"url":"https://example.com","actions":[{"type":"js","script":"localStorage.getItem('x')"}]}`,
		`{"url":"https://example.com","actions":[{"type":"js","script":"eval('alert(1)')"}]}`,
	}
	for _, input := range blockedScripts {
		_, err := tool.Run(input)
		if err == nil {
			t.Errorf("expected block for %s", input)
		}
		if err != nil && !strings.Contains(err.Error(), "bloqueado") && !strings.Contains(err.Error(), "Chrome") {
			t.Errorf("unexpected error: %q", err.Error())
		}
	}
}

func TestBrowserTool_JSEmptyScript(t *testing.T) {
	tool := NewBrowserTool(nil, llm.ClientConfig{})
	_, err := tool.Run(`{"url":"https://example.com","actions":[{"type":"js","script":""}]}`)
	if err == nil {
		t.Fatal("expected error for empty script")
	}
}

func TestBrowserTool_ActionMissingSelector(t *testing.T) {
	types := []string{"wait", "click", "fill", "text"}
	tool := NewBrowserTool(nil, llm.ClientConfig{})
	for _, typ := range types {
		_, err := tool.Run(`{"url":"https://example.com","actions":[{"type":"` + typ + `"}]}`)
		if err == nil {
			t.Errorf("expected error for %s without selector", typ)
		}
		if err != nil && !strings.Contains(err.Error(), "selector") && !strings.Contains(err.Error(), "Chrome") {
			t.Errorf("%s: unexpected error: %q", typ, err.Error())
		}
	}
}
