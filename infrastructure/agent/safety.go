package agent

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

// ErrRequiresConfirmation indica que a tool precisa de confirmacao do usuario
var ErrRequiresConfirmation = errors.New("requires confirmation")

// SafetyGate valida seguranca de tools antes da execucao
type SafetyGate struct {
	log *zap.Logger
}

// NewSafetyGate cria um safety gate
func NewSafetyGate(log *zap.Logger) *SafetyGate {
	return &SafetyGate{log: log.Named("agent.safety")}
}

// Check verifica se a tool pode ser executada
func (g *SafetyGate) Check(tool domain.Tool, input string) error {
	safety := g.getToolSafety(tool)

	switch safety {
	case domain.ToolSafe:
		return nil

	case domain.ToolRestricted:
		if err := g.validateRestricted(tool, input); err != nil {
			g.log.Debug("restricted tool blocked", zap.String("tool", tool.Name()), zap.Error(err))
			return fmt.Errorf("safety: %w", err)
		}
		return nil

	case domain.ToolDangerous:
		g.log.Debug("dangerous tool requires confirmation", zap.String("tool", tool.Name()))
		return fmt.Errorf("%w: %s wants to execute: %s", ErrRequiresConfirmation, tool.Name(), TruncateWithEllipsis(input, 100))
	}

	return nil
}

// getToolSafety retorna o nivel de seguranca da tool
func (g *SafetyGate) getToolSafety(tool domain.Tool) domain.ToolSafety {
	if safeTool, ok := tool.(domain.SafeTool); ok {
		return safeTool.Safety()
	}
	return domain.ToolSafe
}

// GetToolSafety retorna o nivel de seguranca da tool (publico)
func (g *SafetyGate) GetToolSafety(tool domain.Tool) domain.ToolSafety {
	return g.getToolSafety(tool)
}

// validateRestricted valida input de tools com restricao
func (g *SafetyGate) validateRestricted(tool domain.Tool, input string) error {
	switch tool.Name() {
	case "http":
		return g.validateHTTPInput(input)
	case "file_write":
		// file_write ja valida path internamente via safePath()
		return nil
	}
	return nil
}

// validateHTTPInput bloqueia URLs para IPs internos/localhost
func (g *SafetyGate) validateHTTPInput(input string) error {
	// Extrair URL do input (pode ser JSON ou URL direto)
	rawURL := input
	if strings.HasPrefix(strings.TrimSpace(input), "{") {
		// JSON input — nao validar aqui, a tool valida
		return nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("url invalida: %w", err)
	}

	host := parsed.Hostname()
	if host == "" {
		return nil
	}

	// Bloquear localhost e IPs internos
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "0.0.0.0" {
		return fmt.Errorf("localhost access blocked")
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return fmt.Errorf("internal IP access blocked: %s", host)
		}
	}

	return nil
}

