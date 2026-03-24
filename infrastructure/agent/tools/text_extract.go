package tools

import (
	"regexp"
	"strings"

	"github.com/renesul/ok/domain"
)

var (
	tagRegex      = regexp.MustCompile(`<[^>]*>`)
	spaceRegex    = regexp.MustCompile(`\s+`)
	scriptRegex   = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleRegex    = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	noscriptRegex = regexp.MustCompile(`(?is)<noscript[^>]*>.*?</noscript>`)
)

type TextExtractTool struct{}

func (t *TextExtractTool) Name() string        { return "text_extract" }
func (t *TextExtractTool) Description() string { return "extrai texto limpo de HTML (remove tags, scripts, styles)" }
func (t *TextExtractTool) Safety() domain.ToolSafety          { return domain.ToolSafe }

func (t *TextExtractTool) Run(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	// Remove script, style, noscript blocks
	text := scriptRegex.ReplaceAllString(input, "")
	text = styleRegex.ReplaceAllString(text, "")
	text = noscriptRegex.ReplaceAllString(text, "")

	// Remove all HTML tags
	text = tagRegex.ReplaceAllString(text, " ")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")

	// Collapse whitespace
	text = spaceRegex.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	if len(text) > 2000 {
		text = text[:2000] + "..."
	}

	return text, nil
}
