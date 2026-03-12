package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	"ok/internal/config"
	"ok/internal/logger"
)

// Manager manages MCP server connections.
type Manager struct {
	servers map[string]*ServerConnection
	mu      sync.RWMutex
}

// ServerConnection represents a connection to an MCP server.
type ServerConnection struct {
	Config    config.MCPServerConfig
	Client    mcpclient.MCPClient
	Tools     []mcp.Tool
	Connected bool
}

// NewManager creates a new MCP Manager.
func NewManager() *Manager {
	return &Manager{servers: make(map[string]*ServerConnection)}
}

// LoadFromConfig connects to all enabled MCP servers.
func (m *Manager) LoadFromConfig(ctx context.Context, configs []config.MCPServerConfig) error {
	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}
		if err := m.ConnectServer(ctx, cfg); err != nil {
			// Non-fatal: log error, skip server, continue
			logger.ErrorCF("mcp", "Failed to connect MCP server", map[string]any{
				"server": cfg.Name,
				"error":  err.Error(),
			})
		}
	}
	return nil
}

// ConnectServer connects to a single MCP server and discovers its tools.
func (m *Manager) ConnectServer(ctx context.Context, cfg config.MCPServerConfig) error {
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var client *mcpclient.Client
	var err error

	switch cfg.Transport {
	case "stdio":
		// Convert map[string]string to []string ("KEY=VALUE") format
		var envSlice []string
		for k, v := range cfg.Env {
			envSlice = append(envSlice, k+"="+v)
		}
		client, err = mcpclient.NewStdioMCPClient(cfg.Command, envSlice, cfg.Args...)
		if err != nil {
			return fmt.Errorf("failed to create stdio client for %s: %w", cfg.Name, err)
		}
	case "http", "sse":
		opts := []transport.ClientOption{}
		if len(cfg.Headers) > 0 {
			opts = append(opts, transport.WithHeaders(cfg.Headers))
		}
		client, err = mcpclient.NewSSEMCPClient(cfg.URL, opts...)
		if err != nil {
			return fmt.Errorf("failed to create HTTP/SSE client for %s: %w", cfg.Name, err)
		}
		// SSE transport needs explicit Start before Initialize
		if err = client.Start(ctx); err != nil {
			client.Close()
			return fmt.Errorf("failed to start SSE client for %s: %w", cfg.Name, err)
		}
	default:
		return fmt.Errorf("unsupported transport %q for MCP server %s", cfg.Transport, cfg.Name)
	}

	// Initialize the client
	initReq := mcp.InitializeRequest{}
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "ok",
		Version: "1.0.0",
	}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	_, err = client.Initialize(ctx, initReq)
	if err != nil {
		client.Close()
		return fmt.Errorf("failed to initialize MCP server %s: %w", cfg.Name, err)
	}

	// Discover tools
	toolsResult, err := client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		client.Close()
		return fmt.Errorf("failed to list tools from MCP server %s: %w", cfg.Name, err)
	}

	m.mu.Lock()
	m.servers[cfg.Name] = &ServerConnection{
		Config:    cfg,
		Client:    client,
		Tools:     toolsResult.Tools,
		Connected: true,
	}
	m.mu.Unlock()

	logger.InfoCF("mcp", "Connected MCP server", map[string]any{
		"server":    cfg.Name,
		"transport": cfg.Transport,
		"tools":     len(toolsResult.Tools),
	})

	return nil
}

// CallTool calls a tool on a connected MCP server.
func (m *Manager) CallTool(ctx context.Context, serverName, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	m.mu.RLock()
	conn, ok := m.servers[serverName]
	m.mu.RUnlock()

	if !ok || !conn.Connected {
		return nil, fmt.Errorf("MCP server %q not connected", serverName)
	}

	timeout := time.Duration(conn.Config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = toolName
	req.Params.Arguments = args

	result, err := conn.Client.CallTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("MCP tool call %s/%s failed: %w", serverName, toolName, err)
	}

	return result, nil
}

// GetAllTools returns all discovered tools grouped by server name.
func (m *Manager) GetAllTools() map[string][]mcp.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]mcp.Tool)
	for name, conn := range m.servers {
		if conn.Connected {
			result[name] = conn.Tools
		}
	}
	return result
}

// GetServerConfig returns the config for a named server.
func (m *Manager) GetServerConfig(name string) (config.MCPServerConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, ok := m.servers[name]
	if !ok {
		return config.MCPServerConfig{}, false
	}
	return conn.Config, true
}

// Close disconnects all MCP servers.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, conn := range m.servers {
		if conn.Client != nil {
			if err := conn.Client.Close(); err != nil {
				logger.WarnCF("mcp", "Failed to close MCP server", map[string]any{
					"server": name,
					"error":  err.Error(),
				})
			}
		}
		conn.Connected = false
	}
	return nil
}

// TestConnection connects to an MCP server temporarily and returns its tools.
// The connection is closed after discovery.
func TestConnection(ctx context.Context, cfg config.MCPServerConfig) ([]mcp.Tool, error) {
	mgr := NewManager()
	cfg.Enabled = true
	if err := mgr.ConnectServer(ctx, cfg); err != nil {
		return nil, err
	}
	defer mgr.Close()

	tools := mgr.GetAllTools()
	return tools[cfg.Name], nil
}
