package tools

import (
	"context"
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
)

const (
	browserTimeout  = 15 * time.Second
	maxBrowserWords = 5000
)

type BrowserTool struct{}

func NewBrowserTool() *BrowserTool { return &BrowserTool{} }

func (t *BrowserTool) Name() string                       { return "browser" }
func (t *BrowserTool) Description() string                { return "abre pagina web com headless browser e retorna o texto renderizado" }
func (t *BrowserTool) Safety() domain.ToolSafety          { return domain.ToolRestricted }

func (t *BrowserTool) Run(input string) (string, error) {
	return t.RunWithContext(context.Background(), input)
}

func (t *BrowserTool) RunWithContext(ctx context.Context, input string) (string, error) {
	var req struct {
		URL string `json:"url"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("input deve ser JSON: {\"url\":\"https://example.com\"}")
	}

	if req.URL == "" {
		return "", fmt.Errorf("url obrigatorio")
	}

	if err := validateURL(req.URL); err != nil {
		return "", err
	}

	// Tentar headless browser primeiro, fallback para HTTP se Chrome nao existe
	path, exists := launcher.LookPath()
	if exists {
		text, err := fetchWithBrowser(ctx, req.URL, path)
		if err != nil {
			return "", err
		}
		return truncateWords(text, maxBrowserWords), nil
	}

	// Fallback: HTTP simples + strip HTML
	text, err := fetchWithHTTP(ctx, req.URL)
	if err != nil {
		return "", err
	}
	return truncateWords(text, maxBrowserWords), nil
}

func fetchWithBrowser(ctx context.Context, targetURL, chromePath string) (result string, err error) {
	// Safety net: recover de qualquer panic residual do rod
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

	if navErr := page.Context(execCtx).Navigate(targetURL); navErr != nil {
		return "", fmt.Errorf("navegar: %w", navErr)
	}

	if loadErr := page.Context(execCtx).WaitLoad(); loadErr != nil {
		return "", fmt.Errorf("aguardar carregamento: %w", loadErr)
	}

	body, elemErr := page.Element("body")
	if elemErr != nil {
		return "", fmt.Errorf("elemento body: %w", elemErr)
	}

	text, textErr := body.Text()
	if textErr != nil {
		return "", fmt.Errorf("extrair texto: %w", textErr)
	}

	return strings.TrimSpace(text), nil
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB max
	if err != nil {
		return "", fmt.Errorf("ler body: %w", err)
	}

	// Strip HTML tags, scripts, styles
	text := htmlTagsRe.ReplaceAllString(string(body), " ")
	// Collapse whitespace
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

	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			return fmt.Errorf("url interna bloqueada: %s", host)
		}
	}

	return nil
}
