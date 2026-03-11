// OK - Lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OK contributors

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/renesul/ok/pkg/bus"
	"github.com/renesul/ok/pkg/logger"
	"github.com/renesul/ok/pkg/media"
	"github.com/renesul/ok/pkg/providers"
	"github.com/renesul/ok/pkg/tools"
	"github.com/renesul/ok/pkg/utils"
)

// maxToolResultChars caps the content sent back to the LLM from a single tool call.
// Longer outputs are truncated with an indicator showing the original size.
const maxToolResultChars = 4000

// ToolExecutor handles parallel tool execution and result processing.
type ToolExecutor struct {
	bus        *bus.MessageBus
	mediaStore media.MediaStore
}

// ToolExecOptions configures a tool execution batch.
type ToolExecOptions struct {
	Channel      string
	ChatID       string
	SendResponse bool
}

// ToolExecOutput holds the tool result messages to append to the LLM conversation.
type ToolExecOutput struct {
	Messages []providers.Message
}

// ExecuteAll runs tool calls in parallel, processes results, and returns messages for the LLM.
func (te *ToolExecutor) ExecuteAll(
	ctx context.Context,
	agent *AgentInstance,
	toolCalls []providers.ToolCall,
	opts ToolExecOptions,
	iteration int,
) ToolExecOutput {
	type indexedResult struct {
		result *tools.ToolResult
		tc     providers.ToolCall
	}

	results := make([]indexedResult, len(toolCalls))
	var wg sync.WaitGroup

	for i, tc := range toolCalls {
		results[i].tc = tc
		wg.Add(1)
		go func(idx int, tc providers.ToolCall) {
			defer wg.Done()

			argsJSON, _ := json.Marshal(tc.Arguments)
			argsPreview := utils.Truncate(string(argsJSON), 200)
			logger.InfoCF("agent", fmt.Sprintf("Tool call: %s(%s)", tc.Name, argsPreview),
				map[string]any{
					"agent_id":  agent.ID,
					"tool":      tc.Name,
					"iteration": iteration,
				})

			asyncCallback := te.makeAsyncCallback(tc, opts)

			toolResult := agent.Tools.ExecuteWithContext(
				ctx,
				tc.Name,
				tc.Arguments,
				opts.Channel,
				opts.ChatID,
				asyncCallback,
			)
			results[idx].result = toolResult
		}(i, tc)
	}
	wg.Wait()

	// Process results in original order
	var output ToolExecOutput
	for _, r := range results {
		// Send ForUser content to user immediately if not Silent
		if !r.result.Silent && r.result.ForUser != "" && opts.SendResponse {
			te.bus.PublishOutbound(ctx, bus.OutboundMessage{
				Channel: opts.Channel,
				ChatID:  opts.ChatID,
				Content: r.result.ForUser,
			})
			logger.DebugCF("agent", "Sent tool result to user",
				map[string]any{
					"tool":        r.tc.Name,
					"content_len": len(r.result.ForUser),
				})
		}

		// If tool returned media refs, publish them as outbound media
		if len(r.result.Media) > 0 {
			parts := make([]bus.MediaPart, 0, len(r.result.Media))
			for _, ref := range r.result.Media {
				part := bus.MediaPart{Ref: ref}
				if te.mediaStore != nil {
					if _, meta, err := te.mediaStore.ResolveWithMeta(ref); err == nil {
						part.Filename = meta.Filename
						part.ContentType = meta.ContentType
						part.Type = inferMediaType(meta.Filename, meta.ContentType)
					}
				}
				parts = append(parts, part)
			}
			te.bus.PublishOutboundMedia(ctx, bus.OutboundMediaMessage{
				Channel: opts.Channel,
				ChatID:  opts.ChatID,
				Parts:   parts,
			})
		}

		// Build tool result message for LLM
		contentForLLM := r.result.ForLLM
		if contentForLLM == "" && r.result.Err != nil {
			contentForLLM = r.result.Err.Error()
		}

		// Truncate oversized tool outputs to save tokens
		if len(contentForLLM) > maxToolResultChars {
			originalLen := len(contentForLLM)
			contentForLLM = contentForLLM[:maxToolResultChars] +
				fmt.Sprintf("\n...[truncated, %d/%d chars shown]", maxToolResultChars, originalLen)
		}

		output.Messages = append(output.Messages, providers.Message{
			Role:       "tool",
			Content:    contentForLLM,
			ToolCallID: r.tc.ID,
		})
	}

	return output
}

// makeAsyncCallback creates the callback for tools that implement AsyncExecutor.
func (te *ToolExecutor) makeAsyncCallback(
	tc providers.ToolCall,
	opts ToolExecOptions,
) func(context.Context, *tools.ToolResult) {
	return func(_ context.Context, result *tools.ToolResult) {
		// Send ForUser content directly to the user (immediate feedback).
		if !result.Silent && result.ForUser != "" {
			outCtx, outCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer outCancel()
			_ = te.bus.PublishOutbound(outCtx, bus.OutboundMessage{
				Channel: opts.Channel,
				ChatID:  opts.ChatID,
				Content: result.ForUser,
			})
		}

		// Determine content for the agent loop (ForLLM or error).
		content := result.ForLLM
		if content == "" && result.Err != nil {
			content = result.Err.Error()
		}
		if content == "" {
			return
		}

		logger.InfoCF("agent", "Async tool completed, publishing result",
			map[string]any{
				"tool":        tc.Name,
				"content_len": len(content),
				"channel":     opts.Channel,
			})

		pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer pubCancel()
		_ = te.bus.PublishInbound(pubCtx, bus.InboundMessage{
			Channel:  "system",
			SenderID: fmt.Sprintf("async:%s", tc.Name),
			ChatID:   fmt.Sprintf("%s:%s", opts.Channel, opts.ChatID),
			Content:  content,
		})
	}
}
