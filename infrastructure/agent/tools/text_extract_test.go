package tools

import (
	"strings"
	"testing"
)

func TestTextExtract_StripTags(t *testing.T) {
	tool := &TextExtractTool{}
	result, _ := tool.Run("<p>hello</p>")
	if strings.TrimSpace(result) != "hello" {
		t.Fatalf("expected 'hello', got %q", result)
	}
}

func TestTextExtract_StripScripts(t *testing.T) {
	tool := &TextExtractTool{}
	result, _ := tool.Run("<script>alert(1)</script>text")
	if strings.TrimSpace(result) != "text" {
		t.Fatalf("expected 'text', got %q", result)
	}
}

func TestTextExtract_StripStyles(t *testing.T) {
	tool := &TextExtractTool{}
	result, _ := tool.Run("<style>.x{color:red}</style>content")
	if strings.TrimSpace(result) != "content" {
		t.Fatalf("expected 'content', got %q", result)
	}
}

func TestTextExtract_DecodeEntities(t *testing.T) {
	tool := &TextExtractTool{}
	result, _ := tool.Run("&amp; &lt; &gt; &quot;")
	expected := `& < > "`
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestTextExtract_CollapseWhitespace(t *testing.T) {
	tool := &TextExtractTool{}
	result, _ := tool.Run("a    b\n\nc")
	if result != "a b c" {
		t.Fatalf("expected 'a b c', got %q", result)
	}
}

func TestTextExtract_EmptyInput(t *testing.T) {
	tool := &TextExtractTool{}
	result, _ := tool.Run("")
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestTextExtract_Truncation(t *testing.T) {
	tool := &TextExtractTool{}
	input := strings.Repeat("a", 5000)
	result, _ := tool.Run(input)
	if len(result) > 2003 { // 2000 + "..."
		t.Fatalf("expected truncated output <= 2003, got %d", len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Fatalf("expected '...' suffix, got %q", result[len(result)-10:])
	}
}
