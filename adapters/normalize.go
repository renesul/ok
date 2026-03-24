package adapters

import (
	"fmt"
	"strings"

	"github.com/renesul/ok/domain"
)

func NormalizeResponse(resp domain.AgentResponse) string {
	var parts []string

	for _, step := range resp.Steps {
		parts = append(parts, fmt.Sprintf("[%s] %s → %s", step.Tool, step.Name, step.Status))
	}

	for _, msg := range resp.Messages {
		parts = append(parts, msg)
	}

	return strings.Join(parts, "\n")
}
