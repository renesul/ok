// OK - Lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OK contributors

package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	events "ok/app/input/bus"
	"ok/app/types"
	"ok/internal/logger"
	"ok/internal/utils"
	"ok/providers"
)

// PlanAgent is the subset of AgentInstance that the planner needs.
type PlanAgent interface {
	GetID() string
	GetModel() string
	GetMaxIterations() int
	GetMaxTokens() int
	GetTemperature() float64
	GetThinkingLevel() types.ThinkingLevel
	GetProvider() providers.LLMProvider
	GetToolDefs() []providers.ToolDefinition
	GetCandidates() []providers.FallbackCandidate
	GetLightCandidates() []providers.FallbackCandidate
	GetRouter() types.ModelRouter
	GetImageModel() string
	GetImageCandidates() []providers.FallbackCandidate
	GetImageProvider() providers.LLMProvider
}

// ToolExec is the tool execution interface used by the planner.
type ToolExec interface {
	ExecuteAll(ctx context.Context, agent PlanAgent, toolCalls []providers.ToolCall, opts ToolExecOptions, iteration int) ToolExecOutput
}

// MemoryOps is the memory interface used by the planner.
type MemoryOps interface {
	SaveToolInteraction(agent PlanAgent, sessionKey string, msg providers.Message)
	ForceCompression(agent PlanAgent, sessionKey string)
	RebuildMessages(agent PlanAgent, sessionKey, channel, chatID string) []providers.Message
}

// ChannelManager is the channel manager interface for reasoning.
type ChannelManager interface {
	GetChannel(name string) (ReasoningChannel, bool)
}

// ReasoningChannel provides the reasoning channel ID.
type ReasoningChannel interface {
	ReasoningChannelID() string
}

// ToolExecOptions configures a tool execution batch.
type ToolExecOptions struct {
	Channel      string
	ChatID       string
	SendResponse bool
}

// ToolExecOutput holds the tool result messages.
type ToolExecOutput struct {
	Messages []providers.Message
}

// Planner runs the LLM decide→tool→observe loop (ReAct pattern).
type Planner struct {
	Fallback       *providers.FallbackChain
	Bus            *events.MessageBus
	ToolExec       ToolExec
	Memory         MemoryOps
	ChannelManager ChannelManager
}

// PlanOptions configures a single planner run.
type PlanOptions struct {
	SessionKey   string
	Channel      string
	ChatID       string
	UserMessage  string
	SendResponse bool
}

// PlanResult holds the outcome of a planner run.
type PlanResult struct {
	Content    string
	Iterations int
}

// Run executes the ReAct loop: call LLM → if tool calls, execute → loop.
func (p *Planner) Run(
	ctx context.Context,
	agent PlanAgent,
	messages []providers.Message,
	opts PlanOptions,
) (PlanResult, error) {
	iteration := 0
	var finalContent string

	activeCandidates, activeModel := p.selectCandidates(agent, opts.UserMessage, messages)

	// Use a dedicated provider instance when the image model was selected.
	activeProvider := agent.GetProvider()
	if agent.GetImageProvider() != nil && agent.GetImageModel() != "" && hasImageMedia(messages) {
		activeProvider = agent.GetImageProvider()
		// For image requests, strip conversation history to avoid contamination from
		// previous responses that may have incorrectly denied vision capability.
		// Keep only: system message + last user message (with image).
		var systemMsg *providers.Message
		var lastUserMsg *providers.Message
		for i := range messages {
			if messages[i].Role == "system" {
				systemMsg = &messages[i]
			} else if messages[i].Role == "user" {
				lastUserMsg = &messages[i]
			}
		}
		if lastUserMsg != nil {
			if strings.TrimSpace(lastUserMsg.Content) == "" && len(lastUserMsg.Media) > 0 {
				lastUserMsg.Content = "Descreva esta imagem em detalhes."
			}
			if systemMsg != nil {
				messages = []providers.Message{*systemMsg, *lastUserMsg}
			} else {
				messages = []providers.Message{*lastUserMsg}
			}
		}
	}

	providerToolDefs := agent.GetToolDefs()

	for iteration < agent.GetMaxIterations() {
		iteration++

		logger.DebugCF("agent", "LLM iteration",
			map[string]any{
				"agent_id":  agent.GetID(),
				"iteration": iteration,
				"max":       agent.GetMaxIterations(),
			})

		logger.DebugCF("agent", "LLM request",
			map[string]any{
				"agent_id":          agent.GetID(),
				"iteration":         iteration,
				"model":             activeModel,
				"messages_count":    len(messages),
				"tools_count":       len(providerToolDefs),
				"max_tokens":        agent.GetMaxTokens(),
				"temperature":       agent.GetTemperature(),
				"system_prompt_len": len(messages[0].Content),
			})

		logger.DebugCF("agent", "Full LLM request",
			map[string]any{
				"iteration":     iteration,
				"messages_json": FormatMessagesForLog(messages),
				"tools_json":    FormatToolsForLog(providerToolDefs),
			})

		var response *providers.LLMResponse
		var err error

		llmOpts := map[string]any{
			"max_tokens":       agent.GetMaxTokens(),
			"temperature":      agent.GetTemperature(),
			"prompt_cache_key": agent.GetID(),
		}
		if agent.GetThinkingLevel() != types.ThinkingOff {
			if tc, ok := agent.GetProvider().(providers.ThinkingCapable); ok && tc.SupportsThinking() {
				llmOpts["thinking_level"] = string(agent.GetThinkingLevel())
			} else {
				logger.WarnCF("agent", "thinking_level is set but current provider does not support it, ignoring",
					map[string]any{"agent_id": agent.GetID(), "thinking_level": string(agent.GetThinkingLevel())})
			}
		}

		callLLM := func() (*providers.LLMResponse, error) {
			if len(activeCandidates) > 1 && p.Fallback != nil {
				fbResult, fbErr := p.Fallback.Execute(
					ctx,
					activeCandidates,
					func(ctx context.Context, provider, model string) (*providers.LLMResponse, error) {
						return activeProvider.Chat(ctx, messages, providerToolDefs, model, llmOpts)
					},
				)
				if fbErr != nil {
					return nil, fbErr
				}
				if fbResult.Provider != "" && len(fbResult.Attempts) > 0 {
					logger.InfoCF(
						"agent",
						fmt.Sprintf("Fallback: succeeded with %s/%s after %d attempts",
							fbResult.Provider, fbResult.Model, len(fbResult.Attempts)+1),
						map[string]any{"agent_id": agent.GetID(), "iteration": iteration},
					)
				}
				return fbResult.Response, nil
			}
			return activeProvider.Chat(ctx, messages, providerToolDefs, activeModel, llmOpts)
		}

		maxRetries := 2
		for retry := 0; retry <= maxRetries; retry++ {
			response, err = callLLM()
			if err == nil {
				break
			}

			errMsg := strings.ToLower(err.Error())

			isTimeoutError := errors.Is(err, context.DeadlineExceeded) ||
				strings.Contains(errMsg, "deadline exceeded") ||
				strings.Contains(errMsg, "client.timeout") ||
				strings.Contains(errMsg, "timed out") ||
				strings.Contains(errMsg, "timeout exceeded")

			isContextError := !isTimeoutError && (strings.Contains(errMsg, "context_length_exceeded") ||
				strings.Contains(errMsg, "context window") ||
				strings.Contains(errMsg, "maximum context length") ||
				strings.Contains(errMsg, "token limit") ||
				strings.Contains(errMsg, "too many tokens") ||
				strings.Contains(errMsg, "max_tokens") ||
				strings.Contains(errMsg, "invalidparameter") ||
				strings.Contains(errMsg, "prompt is too long") ||
				strings.Contains(errMsg, "request too large"))

			if isTimeoutError && retry < maxRetries {
				backoff := time.Duration(retry+1) * 5 * time.Second
				logger.WarnCF("agent", "Timeout error, retrying after backoff", map[string]any{
					"error":   err.Error(),
					"retry":   retry,
					"backoff": backoff.String(),
				})
				time.Sleep(backoff)
				continue
			}

			if isContextError && retry < maxRetries {
				logger.WarnCF(
					"agent",
					"Context window error detected, attempting compression",
					map[string]any{
						"error": err.Error(),
						"retry": retry,
					},
				)

				if retry == 0 && !utils.IsInternalChannel(opts.Channel) {
					p.Bus.PublishOutbound(ctx, events.OutboundMessage{
						Channel: opts.Channel,
						ChatID:  opts.ChatID,
						Content: "Context window exceeded. Compressing history and retrying...",
					})
				}

				p.Memory.ForceCompression(agent, opts.SessionKey)
				messages = p.Memory.RebuildMessages(agent, opts.SessionKey, opts.Channel, opts.ChatID)
				continue
			}
			break
		}

		if err != nil {
			logger.ErrorCF("agent", "LLM call failed",
				map[string]any{
					"agent_id":  agent.GetID(),
					"iteration": iteration,
					"error":     err.Error(),
				})
			return PlanResult{}, fmt.Errorf("LLM call failed after retries: %w", err)
		}

		go p.HandleReasoning(
			ctx,
			response.Reasoning,
			opts.Channel,
			p.TargetReasoningChannelID(opts.Channel),
		)

		logger.DebugCF("agent", "LLM response",
			map[string]any{
				"agent_id":       agent.GetID(),
				"iteration":      iteration,
				"content_chars":  len(response.Content),
				"tool_calls":     len(response.ToolCalls),
				"reasoning":      response.Reasoning,
				"target_channel": p.TargetReasoningChannelID(opts.Channel),
				"channel":        opts.Channel,
			})

		if len(response.ToolCalls) == 0 {
			finalContent = response.Content
			if finalContent == "" && response.ReasoningContent != "" {
				finalContent = response.ReasoningContent
			}
			logger.InfoCF("agent", "LLM response without tool calls (direct answer)",
				map[string]any{
					"agent_id":      agent.GetID(),
					"iteration":     iteration,
					"content_chars": len(finalContent),
				})
			break
		}

		normalizedToolCalls := make([]providers.ToolCall, 0, len(response.ToolCalls))
		for _, tc := range response.ToolCalls {
			normalizedToolCalls = append(normalizedToolCalls, providers.NormalizeToolCall(tc))
		}

		toolNames := make([]string, 0, len(normalizedToolCalls))
		for _, tc := range normalizedToolCalls {
			toolNames = append(toolNames, tc.Name)
		}
		logger.InfoCF("agent", "LLM requested tool calls",
			map[string]any{
				"agent_id":  agent.GetID(),
				"tools":     toolNames,
				"count":     len(normalizedToolCalls),
				"iteration": iteration,
			})

		assistantMsg := providers.Message{
			Role:             "assistant",
			Content:          response.Content,
			ReasoningContent: response.ReasoningContent,
		}
		for _, tc := range normalizedToolCalls {
			argumentsJSON, _ := json.Marshal(tc.Arguments)
			extraContent := tc.ExtraContent
			thoughtSignature := ""
			if tc.Function != nil {
				thoughtSignature = tc.Function.ThoughtSignature
			}

			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, providers.ToolCall{
				ID:   tc.ID,
				Type: "function",
				Name: tc.Name,
				Function: &providers.FunctionCall{
					Name:             tc.Name,
					Arguments:        string(argumentsJSON),
					ThoughtSignature: thoughtSignature,
				},
				ExtraContent:     extraContent,
				ThoughtSignature: thoughtSignature,
			})
		}
		messages = append(messages, assistantMsg)

		p.Memory.SaveToolInteraction(agent, opts.SessionKey, assistantMsg)

		execOutput := p.ToolExec.ExecuteAll(ctx, agent, normalizedToolCalls, ToolExecOptions{
			Channel:      opts.Channel,
			ChatID:       opts.ChatID,
			SendResponse: opts.SendResponse,
		}, iteration)

		for _, toolMsg := range execOutput.Messages {
			messages = append(messages, toolMsg)
			p.Memory.SaveToolInteraction(agent, opts.SessionKey, toolMsg)
		}
	}

	return PlanResult{Content: finalContent, Iterations: iteration}, nil
}

// hasImageMedia returns true if the last user message in the history contains images.
func hasImageMedia(messages []providers.Message) bool {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role == "user" {
			for _, m := range msg.Media {
				if strings.HasPrefix(m, "data:image/") {
					return true
				}
			}
			return false
		}
	}
	return false
}

func (p *Planner) selectCandidates(
	agent PlanAgent,
	userMsg string,
	history []providers.Message,
) (candidates []providers.FallbackCandidate, model string) {
	// Image routing: if the current message has images and an image model is configured, use it.
	if agent.GetImageModel() != "" && len(agent.GetImageCandidates()) > 0 {
		if hasImageMedia(history) {
			imageCandidates := agent.GetImageCandidates()
			imageModelID := imageCandidates[0].Model
			logger.InfoCF("agent", "Model routing: image model selected",
				map[string]any{
					"agent_id":    agent.GetID(),
					"image_model": agent.GetImageModel(),
					"model_id":    imageModelID,
				})
			return imageCandidates, imageModelID
		}
	}

	router := agent.GetRouter()
	if router == nil || len(agent.GetLightCandidates()) == 0 {
		return agent.GetCandidates(), agent.GetModel()
	}

	_, usedLight, score := router.SelectModel(userMsg, history, agent.GetModel())
	if !usedLight {
		logger.DebugCF("agent", "Model routing: primary model selected",
			map[string]any{
				"agent_id":  agent.GetID(),
				"score":     score,
				"threshold": router.Threshold(),
			})
		return agent.GetCandidates(), agent.GetModel()
	}

	logger.InfoCF("agent", "Model routing: light model selected",
		map[string]any{
			"agent_id":    agent.GetID(),
			"light_model": router.LightModel(),
			"score":       score,
			"threshold":   router.Threshold(),
		})
	return agent.GetLightCandidates(), router.LightModel()
}

// TargetReasoningChannelID returns the reasoning channel ID for a channel.
func (p *Planner) TargetReasoningChannelID(channelName string) string {
	if p.ChannelManager == nil {
		return ""
	}
	if ch, ok := p.ChannelManager.GetChannel(channelName); ok {
		return ch.ReasoningChannelID()
	}
	return ""
}

// HandleReasoning publishes reasoning content to the appropriate channel.
func (p *Planner) HandleReasoning(
	ctx context.Context,
	reasoningContent, channelName, channelID string,
) {
	if reasoningContent == "" || channelName == "" || channelID == "" {
		return
	}

	if ctx.Err() != nil {
		return
	}

	pubCtx, pubCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pubCancel()

	if err := p.Bus.PublishOutbound(pubCtx, events.OutboundMessage{
		Channel: channelName,
		ChatID:  channelID,
		Content: reasoningContent,
	}); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
			errors.Is(err, events.ErrBusClosed) {
			logger.DebugCF("agent", "Reasoning publish skipped (timeout/cancel)", map[string]any{
				"channel": channelName,
				"error":   err.Error(),
			})
		} else {
			logger.WarnCF("agent", "Failed to publish reasoning (best-effort)", map[string]any{
				"channel": channelName,
				"error":   err.Error(),
			})
		}
	}
}

// FormatMessagesForLog formats messages for logging.
func FormatMessagesForLog(messages []providers.Message) string {
	if len(messages) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	for i, msg := range messages {
		fmt.Fprintf(&sb, "  [%d] Role: %s\n", i, msg.Role)
		if len(msg.ToolCalls) > 0 {
			sb.WriteString("  ToolCalls:\n")
			for _, tc := range msg.ToolCalls {
				fmt.Fprintf(&sb, "    - ID: %s, Type: %s, Name: %s\n", tc.ID, tc.Type, tc.Name)
				if tc.Function != nil {
					fmt.Fprintf(
						&sb,
						"      Arguments: %s\n",
						utils.Truncate(tc.Function.Arguments, 200),
					)
				}
			}
		}
		if msg.Content != "" {
			content := utils.Truncate(msg.Content, 200)
			fmt.Fprintf(&sb, "  Content: %s\n", content)
		}
		if msg.ToolCallID != "" {
			fmt.Fprintf(&sb, "  ToolCallID: %s\n", msg.ToolCallID)
		}
		sb.WriteString("\n")
	}
	sb.WriteString("]")
	return sb.String()
}

// FormatToolsForLog formats tool definitions for logging.
func FormatToolsForLog(toolDefs []providers.ToolDefinition) string {
	if len(toolDefs) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	for i, tool := range toolDefs {
		fmt.Fprintf(&sb, "  [%d] Type: %s, Name: %s\n", i, tool.Type, tool.Function.Name)
		fmt.Fprintf(&sb, "      Description: %s\n", tool.Function.Description)
		if len(tool.Function.Parameters) > 0 {
			fmt.Fprintf(
				&sb,
				"      Parameters: %s\n",
				utils.Truncate(fmt.Sprintf("%v", tool.Function.Parameters), 200),
			)
		}
	}
	sb.WriteString("]")
	return sb.String()
}
