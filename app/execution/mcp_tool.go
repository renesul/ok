package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// MCPManager is the interface for calling MCP tools (for testability).
type MCPManager interface {
	CallTool(ctx context.Context, serverName, toolName string, args map[string]any) (*mcp.CallToolResult, error)
}

// MCPTool wraps a single MCP server tool as an agent tool.
type MCPTool struct {
	manager    MCPManager
	serverName string
	prefix     string
	tool       mcp.Tool
}

// NewMCPTool creates a new MCP tool wrapper.
func NewMCPTool(manager MCPManager, serverName, prefix string, tool mcp.Tool) *MCPTool {
	return &MCPTool{
		manager:    manager,
		serverName: serverName,
		prefix:     prefix,
		tool:       tool,
	}
}

// Name returns the tool name, optionally prefixed.
func (t *MCPTool) Name() string {
	name := t.tool.Name
	if t.prefix != "" {
		name = t.prefix + "_" + name
	}
	return name
}

// Description returns the tool description with MCP server info.
func (t *MCPTool) Description() string {
	return fmt.Sprintf("[MCP:%s] %s", t.serverName, t.tool.Description)
}

// Parameters returns the JSON Schema for the tool's input parameters.
func (t *MCPTool) Parameters() map[string]any {
	// Convert ToolInputSchema to map[string]any via JSON round-trip
	data, err := json.Marshal(t.tool.InputSchema)
	if err != nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	return result
}

// Execute calls the MCP tool on the remote server.
func (t *MCPTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	result, err := t.manager.CallTool(ctx, t.serverName, t.tool.Name, args)
	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("MCP tool error: %v", err)}
	}

	// Build text content from result
	var parts []string
	for _, content := range result.Content {
		if tc, ok := mcp.AsTextContent(content); ok {
			parts = append(parts, tc.Text)
		}
	}

	text := strings.Join(parts, "\n")
	if text == "" {
		text = "Tool executed successfully (no output)"
	}

	return &ToolResult{ForLLM: text}
}
