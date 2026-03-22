package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/renesul/ok/domain"
)

func TestCreateUser(t *testing.T) {
	cleanupUsers(t)

	body := `{"name":"John Doe","email":"john@example.com"}`
	req := httptest.NewRequest("POST", "/api/users/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	var user domain.User
	json.NewDecoder(resp.Body).Decode(&user)

	if user.Name != "John Doe" {
		t.Errorf("expected name 'John Doe', got '%s'", user.Name)
	}
	if user.Email != "john@example.com" {
		t.Errorf("expected email 'john@example.com', got '%s'", user.Email)
	}
	if user.ID == 0 {
		t.Error("expected user ID to be set")
	}
}

func TestGetUser(t *testing.T) {
	cleanupUsers(t)

	created := createTestUser(t, "Jane Doe", "jane@example.com")

	req := httptest.NewRequest("GET", "/api/users/"+idToString(created.ID), nil)
	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var user domain.User
	json.NewDecoder(resp.Body).Decode(&user)

	if user.Name != "Jane Doe" {
		t.Errorf("expected name 'Jane Doe', got '%s'", user.Name)
	}
}

func TestListUsers(t *testing.T) {
	cleanupUsers(t)

	createTestUser(t, "User One", "one@example.com")
	createTestUser(t, "User Two", "two@example.com")

	req := httptest.NewRequest("GET", "/api/users/", nil)
	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var users []domain.User
	json.NewDecoder(resp.Body).Decode(&users)

	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestUpdateUser(t *testing.T) {
	cleanupUsers(t)

	user := createTestUser(t, "Old Name", "old@example.com")

	body := `{"name":"New Name","email":"new@example.com"}`
	req := httptest.NewRequest("PUT", "/api/users/"+idToString(user.ID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var updated domain.User
	json.NewDecoder(resp.Body).Decode(&updated)

	if updated.Name != "New Name" {
		t.Errorf("expected name 'New Name', got '%s'", updated.Name)
	}
	if updated.Email != "new@example.com" {
		t.Errorf("expected email 'new@example.com', got '%s'", updated.Email)
	}
}

func TestDeleteUser(t *testing.T) {
	cleanupUsers(t)

	user := createTestUser(t, "Delete Me", "delete@example.com")

	req := httptest.NewRequest("DELETE", "/api/users/"+idToString(user.ID), nil)
	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// confirm deleted
	getReq := httptest.NewRequest("GET", "/api/users/"+idToString(user.ID), nil)
	getResp, _ := testApp.Test(getReq)
	if getResp.StatusCode != 404 {
		t.Fatalf("expected 404 after delete, got %d", getResp.StatusCode)
	}
}

func TestCreateUserValidation(t *testing.T) {
	cleanupUsers(t)

	body := `{"name":"","email":"invalid"}`
	req := httptest.NewRequest("POST", "/api/users/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 422 {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestGetUserNotFound(t *testing.T) {
	cleanupUsers(t)

	req := httptest.NewRequest("GET", "/api/users/999", nil)
	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func createTestUser(t *testing.T, name, email string) domain.User {
	t.Helper()

	body, _ := json.Marshal(map[string]string{"name": name, "email": email})
	req := httptest.NewRequest("POST", "/api/users/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("create test user failed: %v", err)
	}
	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("create test user: expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	var user domain.User
	json.NewDecoder(resp.Body).Decode(&user)
	return user
}

func idToString(id uint) string {
	return fmt.Sprintf("%d", id)
}
