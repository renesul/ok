package tools

import "testing"

func TestGCalManager_Metadata(t *testing.T) {
	tool := NewGCalManagerTool(nil)
	if tool.Name() != "gcal_manager" {
		t.Fatalf("expected name 'gcal_manager', got %q", tool.Name())
	}
	if tool.Safety() != "restricted" {
		t.Fatalf("expected safety 'restricted', got %q", tool.Safety())
	}
}
