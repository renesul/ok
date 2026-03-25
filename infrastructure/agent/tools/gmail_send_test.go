package tools

import "testing"

func TestGmailSend_Metadata(t *testing.T) {
	tool := NewGmailSendTool(nil)
	if tool.Name() != "gmail_send" {
		t.Fatalf("expected name 'gmail_send', got %q", tool.Name())
	}
	if tool.Safety() != "restricted" {
		t.Fatalf("expected safety 'restricted', got %q", tool.Safety())
	}
}
