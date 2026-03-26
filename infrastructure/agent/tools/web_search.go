package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/renesul/ok/domain"
)

const (
	webSearchTimeout = 10 * time.Second
	maxWebResults    = 5
	ddgURL           = "https://html.duckduckgo.com/html/"
)

var (
	resultTitleRe   = regexp.MustCompile(`<a[^>]*class="result__a"[^>]*>([^<]+)</a>`)
	resultURLRe     = regexp.MustCompile(`<a[^>]*class="result__a"[^>]*href="([^"]+)"`)
	resultSnippetRe = regexp.MustCompile(`<a[^>]*class="result__snippet"[^>]*>([^<]+)`)
	htmlTagRe       = regexp.MustCompile(`<[^>]+>`)
)

type WebSearchTool struct{}

func NewWebSearchTool() *WebSearchTool { return &WebSearchTool{} }

func (t *WebSearchTool) Name() string { return "web_search" }
func (t *WebSearchTool) Description() string {
	return "searches the internet via DuckDuckGo and returns top results. Input JSON: {\"query\":\"how to use websocket in Go\"}. Use to find documentation, error solutions, code examples."
}
func (t *WebSearchTool) Safety() domain.ToolSafety { return domain.ToolRestricted }

func (t *WebSearchTool) Run(input string) (string, error) {
	return t.RunWithContext(context.Background(), input)
}

func (t *WebSearchTool) RunWithContext(ctx context.Context, input string) (string, error) {
	query := strings.TrimSpace(input)

	// Aceitar JSON ou string simples
	if strings.HasPrefix(query, "{") {
		// Tentar extrair query do JSON
		if idx := strings.Index(query, "\"query\""); idx >= 0 {
			rest := query[idx:]
			if start := strings.Index(rest, ":"); start >= 0 {
				val := strings.TrimSpace(rest[start+1:])
				val = strings.Trim(val, "\"}")
				val = strings.TrimSpace(val)
				if val != "" {
					query = val
				}
			}
		}
	}

	if query == "" {
		return "", fmt.Errorf("query required")
	}
	if len(query) > 500 {
		return "", fmt.Errorf("query too long (max 500 chars)")
	}

	reqCtx, cancel := context.WithTimeout(ctx, webSearchTimeout)
	defer cancel()

	form := url.Values{"q": {query}}
	req, err := http.NewRequestWithContext(reqCtx, "POST", ddgURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; OK-Agent/1.0)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	html := string(body)
	return parseSearchResults(html, query), nil
}

func parseSearchResults(html, query string) string {
	titles := resultTitleRe.FindAllStringSubmatch(html, maxWebResults*2)
	urls := resultURLRe.FindAllStringSubmatch(html, maxWebResults*2)
	snippets := resultSnippetRe.FindAllStringSubmatch(html, maxWebResults*2)

	var results []string
	count := 0

	for i := 0; i < len(titles) && count < maxWebResults; i++ {
		title := cleanHTML(titles[i][1])
		if title == "" {
			continue
		}

		link := ""
		if i < len(urls) {
			link = extractDDGURL(urls[i][1])
		}

		snippet := ""
		if i < len(snippets) {
			snippet = cleanHTML(snippets[i][1])
		}

		entry := fmt.Sprintf("%d. %s", count+1, title)
		if link != "" {
			entry += "\n   " + link
		}
		if snippet != "" {
			entry += "\n   " + snippet
		}

		results = append(results, entry)
		count++
	}

	if len(results) == 0 {
		return "no results found for: " + query
	}

	return fmt.Sprintf("Results for \"%s\":\n\n%s", query, strings.Join(results, "\n\n"))
}

func extractDDGURL(raw string) string {
	// DuckDuckGo HTML wraps URLs in redirect: //duckduckgo.com/l/?uddg=ENCODED_URL
	if strings.Contains(raw, "uddg=") {
		if idx := strings.Index(raw, "uddg="); idx >= 0 {
			encoded := raw[idx+5:]
			if ampIdx := strings.Index(encoded, "&"); ampIdx >= 0 {
				encoded = encoded[:ampIdx]
			}
			decoded, err := url.QueryUnescape(encoded)
			if err == nil {
				return decoded
			}
		}
	}
	return raw
}

func cleanHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#x27;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return strings.TrimSpace(s)
}
