package orchestrator

import (
	"ok/app/types"
	"ok/providers"
)

// GetID implements planning.PlanAgent.
func (a *AgentInstance) GetID() string { return a.ID }

// GetModel implements planning.PlanAgent.
func (a *AgentInstance) GetModel() string { return a.Model }

// GetMaxIterations implements planning.PlanAgent.
func (a *AgentInstance) GetMaxIterations() int { return a.MaxIterations }

// GetMaxTokens implements planning.PlanAgent.
func (a *AgentInstance) GetMaxTokens() int { return a.MaxTokens }

// GetTemperature implements planning.PlanAgent.
func (a *AgentInstance) GetTemperature() float64 { return a.Temperature }

// GetThinkingLevel implements planning.PlanAgent.
func (a *AgentInstance) GetThinkingLevel() types.ThinkingLevel { return a.ThinkingLevel }

// GetProvider implements planning.PlanAgent.
func (a *AgentInstance) GetProvider() providers.LLMProvider { return a.Provider }

// GetToolDefs implements planning.PlanAgent.
func (a *AgentInstance) GetToolDefs() []providers.ToolDefinition { return a.Tools.ToProviderDefs() }

// GetCandidates implements planning.PlanAgent.
func (a *AgentInstance) GetCandidates() []providers.FallbackCandidate { return a.Candidates }

// GetLightCandidates implements planning.PlanAgent.
func (a *AgentInstance) GetLightCandidates() []providers.FallbackCandidate {
	return a.LightCandidates
}

// GetImageModel implements planning.PlanAgent.
func (a *AgentInstance) GetImageModel() string { return a.ImageModel }

// GetImageCandidates implements planning.PlanAgent.
func (a *AgentInstance) GetImageCandidates() []providers.FallbackCandidate {
	return a.ImageCandidates
}

// GetImageProvider implements planning.PlanAgent.
func (a *AgentInstance) GetImageProvider() providers.LLMProvider {
	return a.ImageProvider
}

// GetRouter implements planning.PlanAgent.
func (a *AgentInstance) GetRouter() types.ModelRouter {
	if a.Router == nil {
		return nil
	}
	return a.Router
}

// GetContextWindow implements output.SummarizeAgent.
func (a *AgentInstance) GetContextWindow() int { return a.ContextWindow }

// GetSummarizeMessageThreshold implements output.SummarizeAgent.
func (a *AgentInstance) GetSummarizeMessageThreshold() int { return a.SummarizeMessageThreshold }

// GetSummarizeTokenPercent implements output.SummarizeAgent.
func (a *AgentInstance) GetSummarizeTokenPercent() int { return a.SummarizeTokenPercent }
