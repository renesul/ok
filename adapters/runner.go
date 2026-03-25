package adapters

import (
	"context"

	"github.com/renesul/ok/domain"
)

// AgentRunner abstracts agent execution for adapter testability.
type AgentRunner interface {
	Run(ctx context.Context, input string) (domain.AgentResponse, error)
}
