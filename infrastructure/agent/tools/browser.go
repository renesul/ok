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
	Path     string `json:"path,omitempty"`      // for screenshot
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
		return "", fmt.Errorf(`input deve ser JSON: {"url":"https://example.com"}`)
	}

	if req.URL == "" {
		return "", fmt.Errorf("url obrigatorio")
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
		return "", fmt.Errorf("actions requerem Chrome/Chromium instalado (headless browser nao encontrado)")
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
		return "", fmt.Errorf("criar pagina: %w", pageErr)
	}
	defer page.Close()

	if navErr := page.Context(execCtx).Navigate(req.URL); navErr != nil {
		return "", fmt.Errorf("navegar: %w", navErr)
	}

	if loadErr := page.Context(execCtx).WaitLoad(); loadErr != nil {
		return "", fmt.Errorf("aguardar carregamento: %w", loadErr)
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
		return "", fmt.Errorf("elemento body: %w", elemErr)
	}

	text, textErr := body.Text()
	if textErr != nil {
		return "", fmt.Errorf("extrair texto: %w", textErr)
	}

	return truncateWords(strings.TrimSpace(text), maxBrowserWords), nil
}

func (t *BrowserTool) executeAction(ctx context.Context, page *rod.Page, action BrowserAction) (string, error) {
	switch action.Type {
	case "wait":
		if action.Selector == "" {
			return "", fmt.Errorf("selector obrigatorio para wait")
		}
		_, err := page.Element(action.Selector)
		return "", err

	case "click":
		if action.Selector == "" {
			return "", fmt.Errorf("selector obrigatorio para click")
		}
		el, err := page.Element(action.Selector)
		if err != nil {
			return "", err
		}
		return "", el.Click(proto.InputMouseButtonLeft, 1)

	case "fill":
		if action.Selector == "" {
			return "", fmt.Errorf("selector obrigatorio para fill")
		}
		el, err := page.Element(action.Selector)
		if err != nil {
			return "", err
		}
		return "", el.Input(action.Value)

	case "js":
		if action.Script == "" {
			return "", fmt.Errorf("script obrigatorio para js")
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
			return "", fmt.Errorf("selector obrigatorio para text")
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
		return "", fmt.Errorf("action type desconhecido: %s", action.Type)
	}
}

var htmlTagsRe = regexp.MustCompile(`<script[^>]*>[\s\S]*?</script>|<style[^>]*>[\s\S]*?</style>|<[^>]+>`)

func fetchWithHTTP(ctx context.Context, targetURL string) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, browserTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("criar request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; OK-Agent/1.0)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return "", fmt.Errorf("ler body: %w", err)
	}

	text := htmlTagsRe.ReplaceAllString(string(body), " ")
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
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
		return fmt.Errorf("url deve comecar com http:// ou https://")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("url invalida: %w", err)
	}

	host := parsed.Hostname()

	if host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" || host == "::1" {
		return fmt.Errorf("url interna bloqueada: %s", host)
	}

	ips, lookupErr := net.LookupIP(host)
	if lookupErr != nil {
		return fmt.Errorf("resolucao DNS falhou (ssrf prevention): %w", lookupErr)
	}

	for _, ip := range ips {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			return fmt.Errorf("url interna bloqueada via resolucao DNS: %s -> %s", host, ip.String())
		}
	}

	return nil
}
