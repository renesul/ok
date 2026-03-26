package engine

import (
	"context"
	"fmt"

	"github.com/renesul/ok/domain"
	agentpkg "github.com/renesul/ok/infrastructure/agent"
	"github.com/renesul/ok/infrastructure/llm"
	"go.uber.org/zap"
)

type ContextManager struct {
	llmConfig         llm.ClientConfig
	buildSystemPrompt func() string
	log               *zap.Logger
}

func NewContextManager(config llm.ClientConfig, promptBuilder func() string, logger *zap.Logger) *ContextManager {
	return &ContextManager{
		llmConfig:         config,
		buildSystemPrompt: promptBuilder,
		log:               logger.Named("context_manager"),
	}
}

// estimateTokens aproxima contagem de tokens via chars/4
func (c *ContextManager) estimateTokens(text string) int {
	return len(text) / 4
}

// PruneContextIfNeeded descarta metade antiga do historico quando o contexto ultrapassa 80% da janela.
func (c *ContextManager) PruneContextIfNeeded(ctx context.Context, state *domain.ExecutionState) {
	maxTokens := c.llmConfig.MaxContextTokens
	if maxTokens <= 0 {
		return
	}

	systemPrompt := c.buildSystemPrompt()
	agentCtx := agentpkg.BuildContext(state)
	totalTokens := c.estimateTokens(systemPrompt) + c.estimateTokens(agentCtx)
	threshold := int(float64(maxTokens) * contextPrunePercent)

	if totalTokens < threshold {
		return
	}

	historyLen := len(state.History)
	if historyLen < 4 {
		return
	}
	splitAt := historyLen / 2

	synthEntry := domain.ExecutionEntry{
		Phase:   domain.PhaseObserve,
		Content: fmt.Sprintf("[Old context pruned — %d entries removed]", splitAt),
	}
	state.History = append([]domain.ExecutionEntry{synthEntry}, state.History[splitAt:]...)

	c.log.Debug("context pruned",
		zap.Int("old_entries", splitAt),
		zap.Int("new_total", len(state.History)),
		zap.Int("tokens_before", totalTokens),
	)
}

// SummarizeIfLong trunca output longo de tools para nao estourar a janela de contexto.
func (c *ContextManager) SummarizeIfLong(ctx context.Context, output string) string {
	const maxOutputLength = 1500
	if len(output) <= maxOutputLength {
		return output
	}

	c.log.Debug("truncating long tool output to prevent window exhaustion", zap.Int("len", len(output)))
	return agentpkg.TruncateWithEllipsis(output, maxOutputLength)
}
