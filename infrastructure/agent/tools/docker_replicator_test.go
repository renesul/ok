package tools

import (
	"strings"
	"testing"
)

func TestDockerReplicator_InvalidJSON(t *testing.T) {
	tool := NewDockerReplicatorTool(nil)
	_, err := tool.Run("not json")
	if err == nil || !strings.Contains(err.Error(), "invalid json") {
		t.Fatalf("expected 'invalid json' error, got %v", err)
	}
}

func TestDockerReplicator_MissingImage(t *testing.T) {
	tool := NewDockerReplicatorTool(nil)
	_, err := tool.Run(`{"cmd":["echo","hi"]}`)
	if err == nil || !strings.Contains(err.Error(), "image") {
		t.Fatalf("expected 'image' error, got %v", err)
	}
}

func TestDockerReplicator_MissingCmd(t *testing.T) {
	tool := NewDockerReplicatorTool(nil)
	_, err := tool.Run(`{"image":"python:3.9"}`)
	if err == nil || !strings.Contains(err.Error(), "cmd") {
		t.Fatalf("expected 'cmd' error, got %v", err)
	}
}

func TestDockerReplicator_Metadata(t *testing.T) {
	tool := NewDockerReplicatorTool(nil)
	if tool.Name() != "docker_replicator" {
		t.Fatalf("expected name 'docker_replicator', got %q", tool.Name())
	}
	if tool.Safety() != "dangerous" {
		t.Fatalf("expected safety 'dangerous', got %q", tool.Safety())
	}
}
