package orchestrator

import (
	"context"
	"testing"

	"ok/internal/config"
	"ok/providers"
)

type mockRegistryProvider struct{}

func (m *mockRegistryProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{Content: "mock", FinishReason: "stop"}, nil
}

func (m *mockRegistryProvider) GetDefaultModel() string {
	return "mock-model"
}

func testCfg() *config.Config {
	return &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         "/tmp/ok-test-registry",
				Model:             "gpt-4",
				MaxTokens:         8192,
				MaxToolIterations: 10,
			},
		},
	}
}

func TestNewAgentRegistry_ImplicitMain(t *testing.T) {
	cfg := testCfg()
	registry := NewAgentRegistry(cfg, &mockRegistryProvider{})

	ids := registry.ListAgentIDs()
	if len(ids) != 1 || ids[0] != "main" {
		t.Errorf("expected implicit main agent, got %v", ids)
	}

	agent, ok := registry.GetAgent("main")
	if !ok || agent == nil {
		t.Fatal("expected to find 'main' agent")
	}
	if agent.ID != "main" {
		t.Errorf("agent.ID = %q, want 'main'", agent.ID)
	}
}

func TestAgentRegistry_GetDefaultAgent(t *testing.T) {
	cfg := testCfg()
	registry := NewAgentRegistry(cfg, &mockRegistryProvider{})

	agent := registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected a default agent")
	}
	if agent.ID != "main" {
		t.Errorf("agent.ID = %q, want 'main'", agent.ID)
	}
}

func TestAgentRegistry_GetAgent_Normalize(t *testing.T) {
	cfg := testCfg()
	registry := NewAgentRegistry(cfg, &mockRegistryProvider{})

	agent, ok := registry.GetAgent("Main")
	if !ok || agent == nil {
		t.Fatal("expected to find agent with normalized ID")
	}
	if agent.ID != "main" {
		t.Errorf("agent.ID = %q, want 'main'", agent.ID)
	}
}
