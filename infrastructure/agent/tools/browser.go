package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/renesul/ok/domain"
	"github.com/renesul/ok/infrastructure/llm"
)

const (
	browserTimeout  = 15 * time.Second
	actionTimeout   = 5 * time.Second
	maxBrowserWords = 5000
)

type BrowserAction struct {
	Type     string `json:"type"`               // wait, click, fill, js, screenshot, text, analyze
	Selector string `json:"selector,omitempty"`
	Value    string `json:"value,omitempty"`     // for fill
	Script   string `json:"script,omitempty"`    // for js
	Prompt   string `json:"prompt,omitempty"`    // for analyze (vision)
}

type browserInput struct {
	URL     string          `json:"url"`
	Actions []BrowserAction `json:"actions,omitempty"`
}

type BrowserTool struct {
	llmClient    *llm.Client
	visionConfig llm.ClientConfig
}

func NewBrowserTool(llmClient *llm.Client, visionConfig llm.ClientConfig) *BrowserTool {
	return &BrowserTool{llmClient: llmClient, visionConfig: visionConfig}
}

func (t *BrowserTool) Name() string { return "browser" }
func (t *BrowserTool) Description() string {
	return `opens web page with headless browser. Input JSON: {"url":"https://...", "actions":[{"type":"click","selector":"#btn"},{"type":"fill","selector":"#email","value":"a@b.com"},{"type":"js","script":"document.title"},{"type":"text","selector":".result"},{"type":"screenshot"},{"type":"analyze","prompt":"describe what you see"},{"type":"wait","selector":"#loaded"}]}. Without actions, returns page text. "analyze" takes a screenshot and uses vision AI to describe what is on the page.`
}
func (t *BrowserTool) Safety() domain.ToolSafety { return domain.ToolRestricted }

func (t *BrowserTool) Run(input string) (string, error) {
	return t.RunWithContext(context.Background(), input)
}

func (t *BrowserTool) RunWithContext(ctx context.Context, input string) (string, error) {
	var req browserInput
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf(`input must be JSON: {"url":"https://example.com"}`)
	}

	if req.URL == "" {
		return "", fmt.Errorf("url required")
	}

	if err := validateURL(req.URL); err != nil {
		return "", err
	}

	path, exists := launcher.LookPath()
	if exists {
		return t.runWithBrowser(ctx, req, path)
	}

	// Fallback HTTP — actions nao suportadas sem browser
	if len(req.Actions) > 0 {
		return "", fmt.Errorf("actions require Chrome/Chromium installed (headless browser not found)")
	}

	text, err := fetchWithHTTP(ctx, req.URL)
	if err != nil {
		return "", err
	}
	return truncateWords(text, maxBrowserWords), nil
}

func (t *BrowserTool) runWithBrowser(ctx context.Context, req browserInput, chromePath string) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("browser panic: %v", r)
		}
	}()

	execCtx, cancel := context.WithTimeout(ctx, browserTimeout)
	defer cancel()

	controlURL, launchErr := launcher.New().Bin(chromePath).Headless(true).Launch()
	if launchErr != nil {
		return "", fmt.Errorf("launch browser: %w", launchErr)
	}

	browser := rod.New().ControlURL(controlURL)
	if connectErr := browser.Connect(); connectErr != nil {
		return "", fmt.Errorf("connect browser: %w", connectErr)
	}
	defer browser.Close()

	page, pageErr := browser.Page(proto.TargetCreateTarget{URL: ""})
	if pageErr != nil {
		return "", fmt.Errorf("create page: %w", pageErr)
	}
	defer page.Close()

	if navErr := page.Context(execCtx).Navigate(req.URL); navErr != nil {
		return "", fmt.Errorf("navigate: %w", navErr)
	}

	if loadErr := page.Context(execCtx).WaitLoad(); loadErr != nil {
		return "", fmt.Errorf("wait for load: %w", loadErr)
	}

	// Execute actions if any
	var outputs []string
	for i, action := range req.Actions {
		actCtx, actCancel := context.WithTimeout(execCtx, actionTimeout)
		out, actErr := t.executeAction(actCtx, page.Context(actCtx), action)
		actCancel()
		if actErr != nil {
			return "", fmt.Errorf("action %d (%s): %w", i+1, action.Type, actErr)
		}
		if out != "" {
			outputs = append(outputs, out)
		}
	}

	// If actions produced output, return that
	if len(outputs) > 0 {
		return strings.Join(outputs, "\n"), nil
	}

	// Default: return body text
	body, elemErr := page.Element("body")
	if elemErr != nil {
		return "", fmt.Errorf("body element: %w", elemErr)
	}

	text, textErr := body.Text()
	if textErr != nil {
		return "", fmt.Errorf("extract text: %w", textErr)
	}

	return truncateWords(strings.TrimSpace(text), maxBrowserWords), nil
}

func (t *BrowserTool) executeAction(ctx context.Context, page *rod.Page, action BrowserAction) (string, error) {
	switch action.Type {
	case "wait":
		if action.Selector == "" {
			return "", fmt.Errorf("selector required for wait")
		}
		_, err := page.Element(action.Selector)
		return "", err

	case "click":
		if action.Selector == "" {
			return "", fmt.Errorf("selector required for click")
		}
		el, err := page.Element(action.Selector)
		if err != nil {
			return "", err
		}
		return "", el.Click(proto.InputMouseButtonLeft, 1)

	case "fill":
		if action.Selector == "" {
			return "", fmt.Errorf("selector required for fill")
		}
		el, err := page.Element(action.Selector)
		if err != nil {
			return "", err
		}
		return "", el.Input(action.Value)

	case "js":
		if action.Script == "" {
			return "", fmt.Errorf("script required for js")
		}
		blocked := []string{"fetch(", "XMLHttpRequest", "document.cookie", "localStorage", "sessionStorage", "eval(", "Function("}
		scriptLower := strings.ToLower(action.Script)
		for _, b := range blocked {
			if strings.Contains(scriptLower, strings.ToLower(b)) {
				return "", fmt.Errorf("script blocked: contains '%s'", b)
			}
		}
		res, err := page.Eval(action.Script)
		if err != nil {
			return "", err
		}
		return res.Value.String(), nil

	case "screenshot":
		data, err := page.Screenshot(true, nil)
		if err != nil {
			return "", err
		}
		encoded := base64.StdEncoding.EncodeToString(data)
		if len(encoded) > 500 {
			return fmt.Sprintf("screenshot captured (%d bytes, base64 truncated)", len(data)), nil
		}
		return encoded, nil

	case "text":
		if action.Selector == "" {
			return "", fmt.Errorf("selector required for text")
		}
		el, err := page.Element(action.Selector)
		if err != nil {
			return "", err
		}
		return el.Text()

	case "analyze":
		if t.llmClient == nil || t.visionConfig.BaseURL == "" {
			return "", fmt.Errorf("vision not configured (set VISION_BASE_URL, VISION_API_KEY, VISION_MODEL)")
		}
		data, err := page.Screenshot(true, nil)
		if err != nil {
			return "", fmt.Errorf("screenshot for vision: %w", err)
		}
		encoded := base64.StdEncoding.EncodeToString(data)
		prompt := action.Prompt
		if prompt == "" {
			prompt = "Describe what you see on this web page. Be concise and factual."
		}
		description, visionErr := t.llmClient.ChatCompletionVision(ctx, t.visionConfig, prompt, encoded)
		if visionErr != nil {
			return "", fmt.Errorf("vision analysis: %w", visionErr)
		}
		return description, nil

	default:
		return "", fmt.Errorf("unknown action type: %s", action.Type)
	}
}

var (
	htmlTagsRe   = regexp.MustCompile(`<script[^>]*>[\s\S]*?</script>|<style[^>]*>[\s\S]*?</style>|<[^>]+>`)
	whitespaceRe = regexp.MustCompile(`\s+`)
)

func fetchWithHTTP(ctx context.Context, targetURL string) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, browserTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; OK-Agent/1.0)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	text := htmlTagsRe.ReplaceAllString(string(body), " ")
	text = whitespaceRe.ReplaceAllString(text, " ")
	return strings.TrimSpace(text), nil
}

func truncateWords(text string, maxWords int) string {
	words := strings.Fields(text)
	if len(words) <= maxWords {
		return text
	}
	return strings.Join(words[:maxWords], " ") + "..."
}

func validateURL(rawURL string) error {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return fmt.Errorf("url must start with http:// or https://")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}

	host := parsed.Hostname()

	if host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" || host == "::1" {
		return fmt.Errorf("internal url blocked: %s", host)
	}

	ips, lookupErr := net.LookupIP(host)
	if lookupErr != nil {
		return fmt.Errorf("DNS resolution failed (ssrf prevention): %w", lookupErr)
	}

	for _, ip := range ips {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			return fmt.Errorf("internal url blocked via DNS resolution: %s -> %s", host, ip.String())
		}
	}

	return nil
}
