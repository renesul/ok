package orchestrator

import (
	"context"

	channels "ok/app/input"
	"ok/app/planning"
	"ok/providers"
)

// toolExecAdapter wraps *ToolExecutor to implement planning.ToolExec.
type toolExecAdapter struct {
	te *ToolExecutor
}

func (a *toolExecAdapter) ExecuteAll(
	ctx context.Context,
	agent planning.PlanAgent,
	toolCalls []providers.ToolCall,
	opts planning.ToolExecOptions,
	iteration int,
) planning.ToolExecOutput {
	return a.te.ExecuteAll(ctx, agent.(*AgentInstance), toolCalls, opts, iteration)
}

// memoryOpsAdapter wraps *MemoryManager to implement planning.MemoryOps.
type memoryOpsAdapter struct {
	mm *MemoryManager
}

func (a *memoryOpsAdapter) SaveToolInteraction(agent planning.PlanAgent, sessionKey string, msg providers.Message) {
	a.mm.SaveToolInteraction(agent.(*AgentInstance), sessionKey, msg)
}

func (a *memoryOpsAdapter) ForceCompression(agent planning.PlanAgent, sessionKey string) {
	a.mm.ForceCompression(agent.(*AgentInstance), sessionKey)
}

func (a *memoryOpsAdapter) RebuildMessages(agent planning.PlanAgent, sessionKey, channel, chatID string) []providers.Message {
	return a.mm.RebuildMessages(agent.(*AgentInstance), sessionKey, channel, chatID)
}

// channelManagerAdapter wraps *channels.Manager to implement planning.ChannelManager.
type channelManagerAdapter struct {
	cm *channels.Manager
}

func (a *channelManagerAdapter) GetChannel(name string) (planning.ReasoningChannel, bool) {
	ch, ok := a.cm.GetChannel(name)
	if !ok {
		return nil, false
	}
	return ch, true
}
