package integration

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

func createTestChatGPTZip(t *testing.T) *bytes.Buffer {
	t.Helper()

	conversations := []map[string]interface{}{
		{
			"title":       "Receita de bolo",
			"create_time": 1700000000.0,
			"update_time": 1700000100.0,
			"mapping": map[string]interface{}{
				"root": map[string]interface{}{
					"id":       "root",
					"message":  nil,
					"parent":   nil,
					"children": []string{"msg1"},
				},
				"msg1": map[string]interface{}{
					"id": "msg1",
					"message": map[string]interface{}{
						"author":      map[string]interface{}{"role": "user"},
						"content":     map[string]interface{}{"content_type": "text", "parts": []string{"Como fazer bolo de chocolate?"}},
						"create_time": 1700000001.0,
					},
					"parent":   "root",
					"children": []string{"msg2"},
				},
				"msg2": map[string]interface{}{
					"id": "msg2",
					"message": map[string]interface{}{
						"author":      map[string]interface{}{"role": "assistant"},
						"content":     map[string]interface{}{"content_type": "text", "parts": []string{"Para fazer um bolo de chocolate voce precisa de farinha, ovos e cacau."}},
						"create_time": 1700000002.0,
					},
					"parent":   "msg1",
					"children": []string{},
				},
			},
		},
		{
			"title":       "Viagem para Roma",
			"create_time": 1700001000.0,
			"update_time": 1700001100.0,
			"mapping": map[string]interface{}{
				"root": map[string]interface{}{
					"id":       "root",
					"message":  nil,
					"parent":   nil,
					"children": []string{"msg1"},
				},
				"msg1": map[string]interface{}{
					"id": "msg1",
					"message": map[string]interface{}{
						"author":      map[string]interface{}{"role": "user"},
						"content":     map[string]interface{}{"content_type": "text", "parts": []string{"Quero visitar Roma em abril"}},
						"create_time": 1700001001.0,
					},
					"parent":   "root",
					"children": []string{"msg2"},
				},
				"msg2": map[string]interface{}{
					"id": "msg2",
					"message": map[string]interface{}{
						"author":      map[string]interface{}{"role": "assistant"},
						"content":     map[string]interface{}{"content_type": "text", "parts": []string{"Roma em abril e otimo! O clima e agradavel."}},
						"create_time": 1700001002.0,
					},
					"parent":   "msg1",
					"children": []string{},
				},
			},
		},
	}

	jsonData, err := json.Marshal(conversations)
	if err != nil {
		t.Fatalf("marshal conversations: %v", err)
	}

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	writer, err := zipWriter.Create("conversations.json")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	writer.Write(jsonData)
	zipWriter.Close()

	return &buf
}

func uploadZip(t *testing.T, zipData *bytes.Buffer) (int, []byte) {
	t.Helper()

	cookie := loginAndGetCookie(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "chatgpt-export.zip")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	part.Write(zipData.Bytes())
	writer.Close()

	req := httptest.NewRequest("POST", "/api/import/chatgpt", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Cookie", "ok_session="+cookie)

	resp, err := testApp.Test(req, -1)
	if err != nil {
		t.Fatalf("import request failed: %v", err)
	}

	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody
}

func TestImportChatGPTSuccess(t *testing.T) {
	defer cleanupAll(t)

	zipData := createTestChatGPTZip(t)
	status, body := uploadZip(t, zipData)

	if status != 201 {
		t.Fatalf("expected 201, got %d: %s", status, string(body))
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	count, ok := result["count"].(float64)
	if !ok || count != 2 {
		t.Fatalf("expected 2 conversations imported, got %v", result["count"])
	}
}

func TestImportChatGPTInvalidFile(t *testing.T) {
	defer cleanupAll(t)

	cookie := loginAndGetCookie(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("file", "data.txt")
	part.Write([]byte("not a zip file"))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/import/chatgpt", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Cookie", "ok_session="+cookie)

	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 400 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestImportChatGPTNoFile(t *testing.T) {
	defer cleanupAll(t)

	cookie := loginAndGetCookie(t)

	req := httptest.NewRequest("POST", "/api/import/chatgpt", nil)
	req.Header.Set("Cookie", "ok_session="+cookie)

	resp, err := testApp.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 400 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(respBody))
	}
}
