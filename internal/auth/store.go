package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ok/internal/utils"
	"ok/internal/logger"
)

var storeMu sync.Mutex

type AuthCredential struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	AccountID    string    `json:"account_id,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Provider     string    `json:"provider"`
	AuthMethod   string    `json:"auth_method"`
	Email        string    `json:"email,omitempty"`
	ProjectID    string    `json:"project_id,omitempty"`
	Label        string    `json:"label,omitempty"`
	APIBase      string    `json:"api_base,omitempty"`
}

type AuthStore struct {
	Credentials map[string]*AuthCredential `json:"credentials"`
}

func (c *AuthCredential) IsExpired() bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(c.ExpiresAt)
}

func (c *AuthCredential) NeedsRefresh() bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().Add(5 * time.Minute).After(c.ExpiresAt)
}

func authFilePath() string {
	if home := os.Getenv("OK_HOME"); home != "" {
		return filepath.Join(home, "auth.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".ok", "auth.json")
	}
	return filepath.Join(home, ".ok", "auth.json")
}

func LoadStore() (*AuthStore, error) {
	path := authFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &AuthStore{Credentials: make(map[string]*AuthCredential)}, nil
		}
		return nil, err
	}

	var store AuthStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	if store.Credentials == nil {
		store.Credentials = make(map[string]*AuthCredential)
	}
	return &store, nil
}

func SaveStore(store *AuthStore) error {
	path := authFilePath()
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	// Use unified atomic write utility with explicit sync for flash storage reliability.
	return utils.WriteFileAtomic(path, data, 0o600)
}

func GetCredential(provider string) (*AuthCredential, error) {
	store, err := LoadStore()
	if err != nil {
		return nil, err
	}
	cred, ok := store.Credentials[provider]
	if !ok {
		return nil, nil
	}
	return cred, nil
}

func SetCredential(provider string, cred *AuthCredential) error {
	storeMu.Lock()
	defer storeMu.Unlock()
	store, err := LoadStore()
	if err != nil {
		return err
	}
	store.Credentials[provider] = cred
	logger.InfoCF("auth", "Credential saved", map[string]any{
		"provider":    provider,
		"auth_method": cred.AuthMethod,
	})
	return SaveStore(store)
}

func DeleteCredential(provider string) error {
	storeMu.Lock()
	defer storeMu.Unlock()
	store, err := LoadStore()
	if err != nil {
		return err
	}
	delete(store.Credentials, provider)
	logger.InfoCF("auth", "Credential deleted", map[string]any{"provider": provider})
	return SaveStore(store)
}

func DeleteAllCredentials() error {
	storeMu.Lock()
	defer storeMu.Unlock()
	path := authFilePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
