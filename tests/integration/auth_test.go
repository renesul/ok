package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
)

func TestLoginSuccess(t *testing.T) {
	defer cleanupSessions(t)

	body := `{"password":"` + testCfg.AuthPassword + `"}`
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["message"] != "login successful" {
		t.Errorf("expected 'login successful', got '%v'", result["message"])
	}
}

func TestLoginSetsCookie(t *testing.T) {
	defer cleanupSessions(t)

	body := `{"password":"` + testCfg.AuthPassword + `"}`
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var found bool
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "ok_session" {
			found = true
			if !cookie.HttpOnly {
				t.Error("session cookie should be HttpOnly")
			}
			if cookie.Value == "" {
				t.Error("session cookie should have a value")
			}
		}
	}
	if !found {
		t.Error("expected ok_session cookie")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	body := `{"password":"wrongpassword"}`
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 401 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestLoginEmptyBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(""))
	req.Header.Set("Content-Type", "application/json")

	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Empty password should be rejected as unauthorized
	if resp.StatusCode != 401 && resp.StatusCode != 422 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401 or 422, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestLogout(t *testing.T) {
	cookie := loginAndGetCookie(t)
	defer cleanupSessions(t)

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.Header.Set("Cookie", "ok_session="+cookie)
	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("logout request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Verify session is invalid
	req = httptest.NewRequest("GET", "/api/conversations", nil)
	req.Header.Set("Cookie", "ok_session="+cookie)
	resp, err = testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401 after logout, got %d", resp.StatusCode)
	}
}

func TestProtectedRouteRedirectsToLogin(t *testing.T) {
	req := httptest.NewRequest("GET", "/chat", nil)
	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 302 {
		t.Fatalf("expected 302 redirect, got %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	if location != "/login" {
		t.Fatalf("expected redirect to /login, got %s", location)
	}
}

func TestProtectedAPIReturns401(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/conversations", nil)
	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
