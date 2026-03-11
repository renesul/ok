package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// mockMCPManager implements MCPManager for testing.
type mockMCPManager struct {
	callToolFunc func(ctx context.Context, serverName, toolName string, args map[string]any) (*mcp.CallToolResult, error)
}

func (m *mockMCPManager) CallTool(ctx context.Context, serverName, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	if m.callToolFunc != nil {
		return m.callToolFunc(ctx, serverName, toolName, args)
	}
	return nil, fmt.Errorf("not implemented")
}

func TestMCPTool_Name(t *testing.T) {
	tool := NewMCPTool(nil, "test-server", "", mcp.Tool{Name: "my_tool"})
	if got := tool.Name(); got != "my_tool" {
		t.Errorf("Name() = %q, want %q", got, "my_tool")
	}

	toolWithPrefix := NewMCPTool(nil, "test-server", "srv", mcp.Tool{Name: "my_tool"})
	if got := toolWithPrefix.Name(); got != "srv_my_tool" {
		t.Errorf("Name() with prefix = %q, want %q", got, "srv_my_tool")
	}
}

func TestMCPTool_Description(t *testing.T) {
	tool := NewMCPTool(nil, "test-server", "", mcp.Tool{
		Name:        "my_tool",
		Description: "Does something useful",
	})
	expected := "[MCP:test-server] Does something useful"
	if got := tool.Description(); got != expected {
		t.Errorf("Description() = %q, want %q", got, expected)
	}
}

func TestMCPTool_Parameters(t *testing.T) {
	tool := NewMCPTool(nil, "test-server", "", mcp.Tool{
		Name: "my_tool",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "search query",
				},
			},
		},
	})
	params := tool.Parameters()
	if params["type"] != "object" {
		t.Errorf("Parameters() type = %v, want 'object'", params["type"])
	}
	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Parameters() properties is not map[string]any")
	}
	if _, ok := props["query"]; !ok {
		t.Error("Parameters() missing 'query' property")
	}
}

func TestMCPTool_Execute_Success(t *testing.T) {
	mgr := &mockMCPManager{
		callToolFunc: func(ctx context.Context, serverName, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
			if serverName != "test-server" {
				t.Errorf("serverName = %q, want %q", serverName, "test-server")
			}
			if toolName != "my_tool" {
				t.Errorf("toolName = %q, want %q", toolName, "my_tool")
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Hello from MCP",
					},
				},
			}, nil
		},
	}

	tool := NewMCPTool(mgr, "test-server", "", mcp.Tool{Name: "my_tool"})
	result := tool.Execute(context.Background(), map[string]any{"query": "test"})
	if result.ForLLM != "Hello from MCP" {
		t.Errorf("Execute() ForLLM = %q, want %q", result.ForLLM, "Hello from MCP")
	}
}

func TestMCPTool_Execute_Error(t *testing.T) {
	mgr := &mockMCPManager{
		callToolFunc: func(ctx context.Context, serverName, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	tool := NewMCPTool(mgr, "test-server", "", mcp.Tool{Name: "my_tool"})
	result := tool.Execute(context.Background(), nil)
	if result.ForLLM != "MCP tool error: connection refused" {
		t.Errorf("Execute() ForLLM = %q, want error message", result.ForLLM)
	}
}

func TestMCPTool_Execute_EmptyResult(t *testing.T) {
	mgr := &mockMCPManager{
		callToolFunc: func(ctx context.Context, serverName, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{},
			}, nil
		},
	}

	tool := NewMCPTool(mgr, "test-server", "", mcp.Tool{Name: "my_tool"})
	result := tool.Execute(context.Background(), nil)
	if result.ForLLM != "Tool executed successfully (no output)" {
		t.Errorf("Execute() ForLLM = %q, want empty result message", result.ForLLM)
	}
}
