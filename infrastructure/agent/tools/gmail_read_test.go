package tools

import "testing"

func TestGmailRead_Metadata(t *testing.T) {
	tool := NewGmailReadTool(nil)
	if tool.Name() != "gmail_read" {
		t.Fatalf("expected name 'gmail_read', got %q", tool.Name())
	}
	if tool.Safety() != "restricted" {
		t.Fatalf("expected safety 'restricted', got %q", tool.Safety())
	}
}
