// OK - Lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OK contributors

package orchestrator

import (
	"context"

	channels "ok/app/input"
	events "ok/app/input/bus"
	"ok/app/planning"
	"ok/providers"
)

// PlanOptions configures a single planner run.
type PlanOptions = planning.PlanOptions

// PlanResult holds the outcome of a planner run.
type PlanResult = planning.PlanResult

// Planner wraps planning.Planner, adapting concrete agent types.
type Planner struct {
	inner          *planning.Planner
	channelManager *channels.Manager
}

func newPlanner(
	fallback *providers.FallbackChain,
	bus *events.MessageBus,
	toolExec *ToolExecutor,
	memMgr *MemoryManager,
) *Planner {
	return &Planner{
		inner: &planning.Planner{
			Fallback: fallback,
			Bus:      bus,
			ToolExec: &toolExecAdapter{te: toolExec},
			Memory:   &memoryOpsAdapter{mm: memMgr},
		},
	}
}

// Run executes the ReAct loop via the planning layer.
func (p *Planner) Run(
	ctx context.Context,
	agent *AgentInstance,
	messages []providers.Message,
	opts PlanOptions,
) (PlanResult, error) {
	return p.inner.Run(ctx, agent, messages, opts)
}

func (p *Planner) setChannelManager(cm *channels.Manager) {
	p.channelManager = cm
	p.inner.ChannelManager = &channelManagerAdapter{cm: cm}
}

func (p *Planner) targetReasoningChannelID(channelName string) string {
	return p.inner.TargetReasoningChannelID(channelName)
}

func (p *Planner) handleReasoning(ctx context.Context, reasoningContent, channelName, channelID string) {
	p.inner.HandleReasoning(ctx, reasoningContent, channelName, channelID)
}
