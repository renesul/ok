package webui

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"ok/internal/auth"
	"ok/internal/config"
	"ok/providers"
)

// providerStatus represents the auth status of a single provider in API responses.
type providerStatus struct {
	Provider   string `json:"provider"`
	AuthMethod string `json:"auth_method"`
	Status     string `json:"status"`
	AccountID  string `json:"account_id,omitempty"`
	Email      string `json:"email,omitempty"`
	ProjectID  string `json:"project_id,omitempty"`
	ExpiresAt  string `json:"expires_at,omitempty"`
	Label      string `json:"label,omitempty"`
	APIBase    string `json:"api_base,omitempty"`
}

// oauthSession stores in-flight OAuth state for browser-based flows.
type oauthSession struct {
	Provider    string
	PKCE        auth.PKCECodes
	State       string
	RedirectURI string
	OAuthCfg    auth.OAuthProviderConfig
	ConfigPath  string
}

// deviceCodeSession stores in-flight device code flow utils.
type deviceCodeSession struct {
	mu         sync.Mutex
	Provider   string
	Info       *auth.DeviceCodeInfo
	OAuthCfg   auth.OAuthProviderConfig
	ConfigPath string
	Status     string // "pending", "success", "error"
	Error      string
	Done       bool
}

var (
	oauthSessions   = map[string]*oauthSession{} // keyed by state
	oauthSessionsMu sync.Mutex

	activeDeviceSession   *deviceCodeSession
	activeDeviceSessionMu sync.Mutex
)

func registerAuthAPI(mux *http.ServeMux, absPath string) {
	mux.HandleFunc("GET /api/auth/status", func(w http.ResponseWriter, r *http.Request) {
		store, err := auth.LoadStore()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load auth store: %v", err), http.StatusInternalServerError)
			return
		}

		result := []providerStatus{}
		for name, cred := range store.Credentials {
			status := "active"
			if cred.IsExpired() {
				status = "expired"
			} else if cred.NeedsRefresh() {
				status = "needs_refresh"
			}
			ps := providerStatus{
				Provider:   name,
				AuthMethod: cred.AuthMethod,
				Status:     status,
				AccountID:  cred.AccountID,
				Email:      cred.Email,
				ProjectID:  cred.ProjectID,
				Label:      cred.Label,
				APIBase:    cred.APIBase,
			}
			if !cred.ExpiresAt.IsZero() {
				ps.ExpiresAt = cred.ExpiresAt.Format(time.RFC3339)
			}
			result = append(result, ps)
		}

		var pendingDevice map[string]any
		activeDeviceSessionMu.Lock()
		if activeDeviceSession != nil {
			activeDeviceSession.mu.Lock()
			pendingDevice = map[string]any{
				"provider": activeDeviceSession.Provider,
				"status":   activeDeviceSession.Status,
			}
			if activeDeviceSession.Info != nil {
				pendingDevice["device_url"] = activeDeviceSession.Info.VerifyURL
				pendingDevice["user_code"] = activeDeviceSession.Info.UserCode
			}
			if activeDeviceSession.Error != "" {
				pendingDevice["error"] = activeDeviceSession.Error
			}
			if activeDeviceSession.Done {
				activeDeviceSession.mu.Unlock()
				activeDeviceSession = nil
			} else {
				activeDeviceSession.mu.Unlock()
			}
		}
		activeDeviceSessionMu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"providers":      result,
			"pending_device": pendingDevice,
		})
	})

	mux.HandleFunc("POST /api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Provider string `json:"provider"`
			Token    string `json:"token,omitempty"`
			APIBase  string `json:"api_base,omitempty"`
			Label    string `json:"label,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		switch req.Provider {
		case "openai":
			if req.Token != "" {
				handleTokenLogin(w, req.Token, absPath, "openai")
			} else {
				handleOpenAILogin(w, absPath)
			}
		case "anthropic":
			handleTokenLogin(w, req.Token, absPath, "anthropic")
		case "google-antigravity", "antigravity":
			handleGoogleAntigravityLogin(w, r, absPath)
		case "groq", "deepseek", "mistral", "xai":
			handleTokenLogin(w, req.Token, absPath, req.Provider)
		default:
			if strings.HasPrefix(req.Provider, "custom-") && req.Token != "" {
				handleCustomProviderLogin(w, req.Token, req.APIBase, req.Label, absPath, req.Provider)
			} else {
				http.Error(w,
					fmt.Sprintf("Unsupported provider: %s", req.Provider),
					http.StatusBadRequest)
			}
		}
	})

	mux.HandleFunc("POST /api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Provider string `json:"provider"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Provider == "" {
			if err := auth.DeleteAllCredentials(); err != nil {
				http.Error(w, fmt.Sprintf("Failed to logout: %v", err), http.StatusInternalServerError)
				return
			}
			clearAllAuthMethodsInConfig(absPath)
		} else {
			if err := auth.DeleteCredential(req.Provider); err != nil {
				http.Error(w, fmt.Sprintf("Failed to logout: %v", err), http.StatusInternalServerError)
				return
			}
			clearAuthMethodInConfig(absPath, req.Provider)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /auth/callback", handleOAuthCallback)
}

func handleOpenAILogin(w http.ResponseWriter, configPath string) {
	activeDeviceSessionMu.Lock()
	if activeDeviceSession != nil {
		activeDeviceSession.mu.Lock()
		if !activeDeviceSession.Done {
			resp := map[string]any{
				"status":     "pending",
				"device_url": activeDeviceSession.Info.VerifyURL,
				"user_code":  activeDeviceSession.Info.UserCode,
				"message":    "Device code flow already in progress. Enter the code in your browser.",
			}
			activeDeviceSession.mu.Unlock()
			activeDeviceSessionMu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		activeDeviceSession.mu.Unlock()
	}
	activeDeviceSessionMu.Unlock()

	oauthCfg := auth.OpenAIOAuthConfig()
	info, err := auth.RequestDeviceCode(oauthCfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to request device code: %v", err), http.StatusInternalServerError)
		return
	}

	session := &deviceCodeSession{
		Provider:   "openai",
		Info:       info,
		OAuthCfg:   oauthCfg,
		ConfigPath: configPath,
		Status:     "pending",
	}

	activeDeviceSessionMu.Lock()
	activeDeviceSession = session
	activeDeviceSessionMu.Unlock()

	go func() {
		deadline := time.After(15 * time.Minute)
		ticker := time.NewTicker(time.Duration(info.Interval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-deadline:
				session.mu.Lock()
				session.Status = "error"
				session.Error = "Authentication timed out after 15 minutes"
				session.Done = true
				session.mu.Unlock()
				return
			case <-ticker.C:
				cred, err := auth.PollDeviceCodeOnce(oauthCfg, info.DeviceAuthID, info.UserCode)
				if err != nil {
					continue
				}
				if cred != nil {
					if saveErr := auth.SetCredential("openai", cred); saveErr != nil {
						session.mu.Lock()
						session.Status = "error"
						session.Error = saveErr.Error()
						session.Done = true
						session.mu.Unlock()
						return
					}
					updateConfigAfterLogin(configPath, "openai", cred)
					session.mu.Lock()
					session.Status = "success"
					session.Done = true
					session.mu.Unlock()
					log.Printf("OpenAI device code login successful (account: %s)", cred.AccountID)
					return
				}
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":     "pending",
		"device_url": info.VerifyURL,
		"user_code":  info.UserCode,
		"message":    "Open the URL and enter the code to authenticate.",
	})
}

func handleTokenLogin(w http.ResponseWriter, token, configPath, provider string) {
	if token == "" {
		http.Error(w, fmt.Sprintf("Token is required for %s login", provider), http.StatusBadRequest)
		return
	}

	cred := &auth.AuthCredential{
		AccessToken: token,
		Provider:    provider,
		AuthMethod:  "token",
	}

	if err := auth.SetCredential(provider, cred); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save credentials: %v", err), http.StatusInternalServerError)
		return
	}

	updateConfigAfterLogin(configPath, provider, cred)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": provider + " token saved",
	})
}

func handleCustomProviderLogin(w http.ResponseWriter, token, apiBase, label, configPath, provider string) {
	if token == "" {
		http.Error(w, "Token is required for custom provider login", http.StatusBadRequest)
		return
	}

	cred := &auth.AuthCredential{
		AccessToken: token,
		Provider:    provider,
		AuthMethod:  "token",
		Label:       label,
		APIBase:     apiBase,
	}

	if err := auth.SetCredential(provider, cred); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save credentials: %v", err), http.StatusInternalServerError)
		return
	}

	// Add a model entry for the custom provider if api_base is specified
	if apiBase != "" {
		updateCustomProviderModel(configPath, provider, label, apiBase)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Custom provider saved",
	})
}

func updateCustomProviderModel(configPath, provider, label, apiBase string) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Printf("Warning: could not load config to update custom provider model: %v", err)
		return
	}

	// Check if a model already exists for this custom provider
	for i := range cfg.ModelList {
		if cfg.ModelList[i].Model != "" && strings.HasPrefix(cfg.ModelList[i].Model, provider+"/") {
			cfg.ModelList[i].AuthMethod = "token"
			cfg.ModelList[i].APIBase = apiBase
			if err := config.SaveConfig(configPath, cfg); err != nil {
				log.Printf("Warning: could not update config: %v", err)
			}
			return
		}
	}
	// Don't auto-create a model entry for custom providers — user will do that via model modal
}

func handleGoogleAntigravityLogin(w http.ResponseWriter, r *http.Request, configPath string) {
	oauthCfg := auth.GoogleAntigravityOAuthConfig()

	pkce, err := auth.GeneratePKCE()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate PKCE: %v", err), http.StatusInternalServerError)
		return
	}

	state, err := auth.GenerateState()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate state: %v", err), http.StatusInternalServerError)
		return
	}

	scheme := "http"
	redirectURI := fmt.Sprintf("%s://%s/auth/callback", scheme, r.Host)

	authURL := auth.BuildAuthorizeURL(oauthCfg, pkce, state, redirectURI)

	oauthSessionsMu.Lock()
	oauthSessions[state] = &oauthSession{
		Provider:    "google-antigravity",
		PKCE:        pkce,
		State:       state,
		RedirectURI: redirectURI,
		OAuthCfg:    oauthCfg,
		ConfigPath:  configPath,
	}
	oauthSessionsMu.Unlock()

	go func() {
		time.Sleep(10 * time.Minute)
		oauthSessionsMu.Lock()
		delete(oauthSessions, state)
		oauthSessionsMu.Unlock()
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "redirect",
		"auth_url": authURL,
		"message":  "Open the URL to authenticate with Google.",
	})
}

func handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	oauthSessionsMu.Lock()
	session, ok := oauthSessions[state]
	if ok {
		delete(oauthSessions, state)
	}
	oauthSessionsMu.Unlock()

	if !ok {
		http.Error(w, "Invalid or expired OAuth state", http.StatusBadRequest)
		return
	}

	if code == "" {
		errMsg := r.URL.Query().Get("error")
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w,
			`<html><body><h2>Authentication failed</h2><p>%s</p><p>You can close this window.</p></body></html>`,
			html.EscapeString(errMsg))
		return
	}

	cred, err := auth.ExchangeCodeForTokens(session.OAuthCfg, code, session.PKCE.CodeVerifier, session.RedirectURI)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w,
			`<html><body><h2>Authentication failed</h2><p>%s</p><p>You can close this window.</p></body></html>`,
			html.EscapeString(err.Error()))
		return
	}

	cred.Provider = session.Provider

	if session.Provider == "google-antigravity" {
		if email, err := fetchGoogleUserEmail(cred.AccessToken); err == nil {
			cred.Email = email
		}
		if projectID, err := providers.FetchAntigravityProjectID(cred.AccessToken); err == nil {
			cred.ProjectID = projectID
		}
	}

	if err := auth.SetCredential(session.Provider, cred); err != nil {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><body><h2>Failed to save credentials</h2><p>%s</p></body></html>`, html.EscapeString(err.Error()))
		return
	}

	updateConfigAfterLogin(session.ConfigPath, session.Provider, cred)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<html><body>
		<h2>Authentication successful!</h2>
		<p>Redirecting back to Config Editor...</p>
		<script>setTimeout(function(){ window.location.href = '/#auth'; }, 1000);</script>
	</body></html>`)
}

func fetchGoogleUserEmail(accessToken string) (string, error) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading userinfo response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("userinfo request failed: %s", string(body))
	}

	var userInfo struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return "", err
	}
	return userInfo.Email, nil
}

// ── Config update helpers ─────────────────────────────────

func updateConfigAfterLogin(configPath, provider string, cred *auth.AuthCredential) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Printf("Warning: could not load config to update auth_method: %v", err)
		return
	}

	switch provider {
	case "openai":
		authMethod := cred.AuthMethod
		if authMethod == "" {
			authMethod = "oauth"
		}
		found := false
		for i := range cfg.ModelList {
			if isOpenAIModel(cfg.ModelList[i].Model) {
				cfg.ModelList[i].AuthMethod = authMethod
				found = true
				break
			}
		}
		if !found {
			cfg.ModelList = append(cfg.ModelList, config.ModelConfig{
				ModelName:  "gpt-5.2",
				Model:      "openai/gpt-5.2",
				AuthMethod: authMethod,
			})
		}
		if cfg.Agents.Defaults.ModelName == "" {
			cfg.Agents.Defaults.ModelName = "gpt-5.2"
		}

	case "anthropic":
		found := false
		for i := range cfg.ModelList {
			if isAnthropicModel(cfg.ModelList[i].Model) {
				cfg.ModelList[i].AuthMethod = "token"
				found = true
				break
			}
		}
		if !found {
			cfg.ModelList = append(cfg.ModelList, config.ModelConfig{
				ModelName:  "claude-sonnet-4.6",
				Model:      "anthropic/claude-sonnet-4.6",
				AuthMethod: "token",
			})
		}
		if cfg.Agents.Defaults.ModelName == "" {
			cfg.Agents.Defaults.ModelName = "claude-sonnet-4.6"
		}

	case "google-antigravity":
		found := false
		for i := range cfg.ModelList {
			if isAntigravityModel(cfg.ModelList[i].Model) {
				cfg.ModelList[i].AuthMethod = "oauth"
				found = true
				break
			}
		}
		if !found {
			cfg.ModelList = append(cfg.ModelList, config.ModelConfig{
				ModelName:  "gemini-flash",
				Model:      "antigravity/gemini-3-flash",
				AuthMethod: "oauth",
			})
		}
		if cfg.Agents.Defaults.ModelName == "" {
			cfg.Agents.Defaults.ModelName = "gemini-flash"
		}

	case "groq":
		updateOrAddTokenModel(cfg, "groq", isGroqModel, "groq-llama", "groq/llama-4-scout-17b-16e-instruct", providers.GetDefaultAPIBase("groq"))

	case "deepseek":
		updateOrAddTokenModel(cfg, "deepseek", isDeepSeekModel, "deepseek-chat", "deepseek/deepseek-chat", providers.GetDefaultAPIBase("deepseek"))

	case "mistral":
		updateOrAddTokenModel(cfg, "mistral", isMistralModel, "mistral-medium", "mistral/mistral-medium-latest", providers.GetDefaultAPIBase("mistral"))

	case "xai":
		updateOrAddTokenModel(cfg, "xai", isXAIModel, "grok-3-mini", "xai/grok-3-mini", providers.GetDefaultAPIBase("xai"))
	}

	if err := config.SaveConfig(configPath, cfg); err != nil {
		log.Printf("Warning: could not update config: %v", err)
	}
}

func clearAuthMethodInConfig(configPath, provider string) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return
	}

	for i := range cfg.ModelList {
		switch provider {
		case "openai":
			if isOpenAIModel(cfg.ModelList[i].Model) {
				cfg.ModelList[i].AuthMethod = ""
			}
		case "anthropic":
			if isAnthropicModel(cfg.ModelList[i].Model) {
				cfg.ModelList[i].AuthMethod = ""
			}
		case "google-antigravity", "antigravity":
			if isAntigravityModel(cfg.ModelList[i].Model) {
				cfg.ModelList[i].AuthMethod = ""
			}
		case "groq":
			if isGroqModel(cfg.ModelList[i].Model) {
				cfg.ModelList[i].AuthMethod = ""
			}
		case "deepseek":
			if isDeepSeekModel(cfg.ModelList[i].Model) {
				cfg.ModelList[i].AuthMethod = ""
			}
		case "mistral":
			if isMistralModel(cfg.ModelList[i].Model) {
				cfg.ModelList[i].AuthMethod = ""
			}
		case "xai":
			if isXAIModel(cfg.ModelList[i].Model) {
				cfg.ModelList[i].AuthMethod = ""
			}
		default:
			if strings.HasPrefix(provider, "custom-") && strings.HasPrefix(cfg.ModelList[i].Model, provider+"/") {
				cfg.ModelList[i].AuthMethod = ""
			}
		}
	}

	if err := config.SaveConfig(configPath, cfg); err != nil {
		log.Printf("Warning: could not update config: %v", err)
	}
}

func clearAllAuthMethodsInConfig(configPath string) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return
	}
	for i := range cfg.ModelList {
		cfg.ModelList[i].AuthMethod = ""
	}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		log.Printf("Warning: could not update config: %v", err)
	}
}

func isOpenAIModel(model string) bool {
	return model == "openai" || strings.HasPrefix(model, "openai/")
}

func isAnthropicModel(model string) bool {
	return model == "anthropic" || strings.HasPrefix(model, "anthropic/")
}

func isAntigravityModel(model string) bool {
	return model == "antigravity" || model == "google-antigravity" ||
		strings.HasPrefix(model, "antigravity/") || strings.HasPrefix(model, "google-antigravity/")
}

func isGroqModel(model string) bool {
	return model == "groq" || strings.HasPrefix(model, "groq/")
}

func isDeepSeekModel(model string) bool {
	return model == "deepseek" || strings.HasPrefix(model, "deepseek/")
}

func isMistralModel(model string) bool {
	return model == "mistral" || strings.HasPrefix(model, "mistral/")
}

func isXAIModel(model string) bool {
	return model == "xai" || strings.HasPrefix(model, "xai/")
}

func updateOrAddTokenModel(cfg *config.Config, provider string, isModel func(string) bool, modelName, model, apiBase string) {
	found := false
	for i := range cfg.ModelList {
		if isModel(cfg.ModelList[i].Model) {
			cfg.ModelList[i].AuthMethod = "token"
			found = true
			break
		}
	}
	if !found {
		cfg.ModelList = append(cfg.ModelList, config.ModelConfig{
			ModelName:  modelName,
			Model:      model,
			APIBase:    apiBase,
			AuthMethod: "token",
		})
	}
	if cfg.Agents.Defaults.ModelName == "" {
		cfg.Agents.Defaults.ModelName = modelName
	}
}
