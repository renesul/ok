// OK - Lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OK contributors

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/renesul/ok/pkg/bus"
	"github.com/renesul/ok/pkg/channels"
	"github.com/renesul/ok/pkg/constants"
	"github.com/renesul/ok/pkg/logger"
	"github.com/renesul/ok/pkg/providers"
)

// Planner runs the LLM decide→tool→observe loop (ReAct pattern).
type Planner struct {
	fallback       *providers.FallbackChain
	bus            *bus.MessageBus
	toolExec       *ToolExecutor
	memory         *MemoryManager
	channelManager *channels.Manager
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
	agent *AgentInstance,
	messages []providers.Message,
	opts PlanOptions,
) (PlanResult, error) {
	iteration := 0
	var finalContent string

	// Select model tier once for the entire turn (sticky across tool iterations).
	activeCandidates, activeModel := p.selectCandidates(agent, opts.UserMessage, messages)

	// Cache tool definitions once — tools don't change during a turn.
	// This avoids rebuilding the slice each iteration and keeps the reference
	// stable for provider-side KV cache reuse.
	providerToolDefs := agent.Tools.ToProviderDefs()

	for iteration < agent.MaxIterations {
		iteration++

		logger.DebugCF("agent", "LLM iteration",
			map[string]any{
				"agent_id":  agent.ID,
				"iteration": iteration,
				"max":       agent.MaxIterations,
			})

		logger.DebugCF("agent", "LLM request",
			map[string]any{
				"agent_id":          agent.ID,
				"iteration":         iteration,
				"model":             activeModel,
				"messages_count":    len(messages),
				"tools_count":       len(providerToolDefs),
				"max_tokens":        agent.MaxTokens,
				"temperature":       agent.Temperature,
				"system_prompt_len": len(messages[0].Content),
			})

		logger.DebugCF("agent", "Full LLM request",
			map[string]any{
				"iteration":     iteration,
				"messages_json": formatMessagesForLog(messages),
				"tools_json":    formatToolsForLog(providerToolDefs),
			})

		// Call LLM with fallback chain if multiple candidates are configured.
		var response *providers.LLMResponse
		var err error

		llmOpts := map[string]any{
			"max_tokens":       agent.MaxTokens,
			"temperature":      agent.Temperature,
			"prompt_cache_key": agent.ID,
		}
		if agent.ThinkingLevel != ThinkingOff {
			if tc, ok := agent.Provider.(providers.ThinkingCapable); ok && tc.SupportsThinking() {
				llmOpts["thinking_level"] = string(agent.ThinkingLevel)
			} else {
				logger.WarnCF("agent", "thinking_level is set but current provider does not support it, ignoring",
					map[string]any{"agent_id": agent.ID, "thinking_level": string(agent.ThinkingLevel)})
			}
		}

		callLLM := func() (*providers.LLMResponse, error) {
			if len(activeCandidates) > 1 && p.fallback != nil {
				fbResult, fbErr := p.fallback.Execute(
					ctx,
					activeCandidates,
					func(ctx context.Context, provider, model string) (*providers.LLMResponse, error) {
						return agent.Provider.Chat(ctx, messages, providerToolDefs, model, llmOpts)
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
						map[string]any{"agent_id": agent.ID, "iteration": iteration},
					)
				}
				return fbResult.Response, nil
			}
			return agent.Provider.Chat(ctx, messages, providerToolDefs, activeModel, llmOpts)
		}

		// Retry loop for context/token errors
		maxRetries := 2
		for retry := 0; retry <= maxRetries; retry++ {
			response, err = callLLM()
			if err == nil {
				break
			}

			errMsg := strings.ToLower(err.Error())

			// Check if this is a network/HTTP timeout — not a context window error.
			isTimeoutError := errors.Is(err, context.DeadlineExceeded) ||
				strings.Contains(errMsg, "deadline exceeded") ||
				strings.Contains(errMsg, "client.timeout") ||
				strings.Contains(errMsg, "timed out") ||
				strings.Contains(errMsg, "timeout exceeded")

			// Detect real context window / token limit errors, excluding network timeouts.
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

				if retry == 0 && !constants.IsInternalChannel(opts.Channel) {
					p.bus.PublishOutbound(ctx, bus.OutboundMessage{
						Channel: opts.Channel,
						ChatID:  opts.ChatID,
						Content: "Context window exceeded. Compressing history and retrying...",
					})
				}

				p.memory.ForceCompression(agent, opts.SessionKey)
				messages = p.memory.RebuildMessages(agent, opts.SessionKey, opts.Channel, opts.ChatID)
				continue
			}
			break
		}

		if err != nil {
			logger.ErrorCF("agent", "LLM call failed",
				map[string]any{
					"agent_id":  agent.ID,
					"iteration": iteration,
					"error":     err.Error(),
				})
			return PlanResult{}, fmt.Errorf("LLM call failed after retries: %w", err)
		}

		go p.handleReasoning(
			ctx,
			response.Reasoning,
			opts.Channel,
			p.targetReasoningChannelID(opts.Channel),
		)

		logger.DebugCF("agent", "LLM response",
			map[string]any{
				"agent_id":       agent.ID,
				"iteration":      iteration,
				"content_chars":  len(response.Content),
				"tool_calls":     len(response.ToolCalls),
				"reasoning":      response.Reasoning,
				"target_channel": p.targetReasoningChannelID(opts.Channel),
				"channel":        opts.Channel,
			})

		// No tool calls → final answer
		if len(response.ToolCalls) == 0 {
			finalContent = response.Content
			if finalContent == "" && response.ReasoningContent != "" {
				finalContent = response.ReasoningContent
			}
			logger.InfoCF("agent", "LLM response without tool calls (direct answer)",
				map[string]any{
					"agent_id":      agent.ID,
					"iteration":     iteration,
					"content_chars": len(finalContent),
				})
			break
		}

		// Normalize tool calls
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
				"agent_id":  agent.ID,
				"tools":     toolNames,
				"count":     len(normalizedToolCalls),
				"iteration": iteration,
			})

		// Build assistant message with tool calls
		assistantMsg := providers.Message{
			Role:             "assistant",
			Content:          response.Content,
			ReasoningContent: response.ReasoningContent,
		}
		for _, tc := range normalizedToolCalls {
			argumentsJSON, _ := json.Marshal(tc.Arguments)
			// Copy ExtraContent to ensure thought_signature is persisted for Gemini 3
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

		// Save assistant message with tool calls to session
		p.memory.SaveToolInteraction(agent, opts.SessionKey, assistantMsg)

		// Execute tools
		execOutput := p.toolExec.ExecuteAll(ctx, agent, normalizedToolCalls, ToolExecOptions{
			Channel:      opts.Channel,
			ChatID:       opts.ChatID,
			SendResponse: opts.SendResponse,
		}, iteration)

		// Append tool results and save to session
		for _, toolMsg := range execOutput.Messages {
			messages = append(messages, toolMsg)
			p.memory.SaveToolInteraction(agent, opts.SessionKey, toolMsg)
		}
	}

	return PlanResult{Content: finalContent, Iterations: iteration}, nil
}

// selectCandidates returns the model candidates and resolved model name for a turn.
// When model routing is configured and the message scores below the complexity
// threshold, it returns the light model candidates instead of the primary ones.
func (p *Planner) selectCandidates(
	agent *AgentInstance,
	userMsg string,
	history []providers.Message,
) (candidates []providers.FallbackCandidate, model string) {
	if agent.Router == nil || len(agent.LightCandidates) == 0 {
		return agent.Candidates, agent.Model
	}

	_, usedLight, score := agent.Router.SelectModel(userMsg, history, agent.Model)
	if !usedLight {
		logger.DebugCF("agent", "Model routing: primary model selected",
			map[string]any{
				"agent_id":  agent.ID,
				"score":     score,
				"threshold": agent.Router.Threshold(),
			})
		return agent.Candidates, agent.Model
	}

	logger.InfoCF("agent", "Model routing: light model selected",
		map[string]any{
			"agent_id":    agent.ID,
			"light_model": agent.Router.LightModel(),
			"score":       score,
			"threshold":   agent.Router.Threshold(),
		})
	return agent.LightCandidates, agent.Router.LightModel()
}

func (p *Planner) targetReasoningChannelID(channelName string) string {
	if p.channelManager == nil {
		return ""
	}
	if ch, ok := p.channelManager.GetChannel(channelName); ok {
		return ch.ReasoningChannelID()
	}
	return ""
}

func (p *Planner) handleReasoning(
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

	if err := p.bus.PublishOutbound(pubCtx, bus.OutboundMessage{
		Channel: channelName,
		ChatID:  channelID,
		Content: reasoningContent,
	}); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
			errors.Is(err, bus.ErrBusClosed) {
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
