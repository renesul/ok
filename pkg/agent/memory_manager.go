// OK - Lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OK contributors

package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/renesul/ok/pkg/logger"
	"github.com/renesul/ok/pkg/media"
	"github.com/renesul/ok/pkg/providers"
	"github.com/renesul/ok/pkg/rag"
)

// MemoryManager handles loading and saving conversation state.
type MemoryManager struct {
	mediaStore   media.MediaStore
	maxMediaSize int
	retriever    *rag.Retriever // optional RAG retriever for indexing interactions
}

// MemoryLoadOptions configures how conversation context is loaded.
type MemoryLoadOptions struct {
	SessionKey  string
	UserMessage string
	Media       []string
	Channel     string
	ChatID      string
	NoHistory   bool
}

// ConversationContext holds the full message list for an LLM call.
type ConversationContext struct {
	Messages []providers.Message
}

// Load builds the full message context for an LLM call.
func (mm *MemoryManager) Load(agent *AgentInstance, opts MemoryLoadOptions) ConversationContext {
	var history []providers.Message
	var summary string
	if !opts.NoHistory {
		history = agent.Sessions.GetHistory(opts.SessionKey)
		summary = agent.Sessions.GetSummary(opts.SessionKey)
	}
	messages := agent.ContextBuilder.BuildMessages(
		history, summary, opts.UserMessage, opts.Media, opts.Channel, opts.ChatID,
	)
	messages = resolveMediaRefs(messages, mm.mediaStore, mm.maxMediaSize)
	return ConversationContext{Messages: messages}
}

// SaveUserMessage records the user's message in session and indexes it for RAG.
func (mm *MemoryManager) SaveUserMessage(agent *AgentInstance, sessionKey, content string) {
	agent.Sessions.AddMessage(sessionKey, "user", content)
	mm.indexForRAG(content, "user", sessionKey)
}

// SaveAssistantMessage records the assistant's response, persists the session, and indexes for RAG.
func (mm *MemoryManager) SaveAssistantMessage(agent *AgentInstance, sessionKey, content string) {
	agent.Sessions.AddMessage(sessionKey, "assistant", content)
	agent.Sessions.Save(sessionKey)
	mm.indexForRAG(content, "assistant", sessionKey)
}

// indexForRAG asynchronously indexes an interaction for RAG retrieval.
func (mm *MemoryManager) indexForRAG(content, role, sessionKey string) {
	if mm.retriever == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		interaction := rag.Interaction{
			ID:         fmt.Sprintf("%s:%d", sessionKey, time.Now().UnixNano()),
			Role:       role,
			Content:    content,
			SessionKey: sessionKey,
			Timestamp:  time.Now(),
		}
		if err := mm.retriever.Index(ctx, interaction); err != nil {
			logger.WarnCF("rag", "Failed to index interaction", map[string]any{
				"role":  role,
				"error": err.Error(),
			})
		}
	}()
}

// SaveToolInteraction records an assistant or tool message during the loop.
func (mm *MemoryManager) SaveToolInteraction(agent *AgentInstance, sessionKey string, msg providers.Message) {
	agent.Sessions.AddFullMessage(sessionKey, msg)
}

// ForceCompression aggressively reduces context when the limit is hit.
// It drops the oldest 50% of messages (keeping system prompt and last user message).
func (mm *MemoryManager) ForceCompression(agent *AgentInstance, sessionKey string) {
	history := agent.Sessions.GetHistory(sessionKey)
	if len(history) <= 4 {
		return
	}

	conversation := history[1 : len(history)-1]
	if len(conversation) == 0 {
		return
	}

	mid := len(conversation) / 2
	droppedCount := mid
	keptConversation := conversation[mid:]

	newHistory := make([]providers.Message, 0, 1+len(keptConversation)+1)

	// Append compression note to the original system prompt instead of adding a new system message.
	// This avoids having two consecutive system messages which some APIs (like Zhipu) reject.
	compressionNote := fmt.Sprintf(
		"\n\n[System Note: Emergency compression dropped %d oldest messages due to context limit]",
		droppedCount,
	)
	enhancedSystemPrompt := history[0]
	enhancedSystemPrompt.Content = enhancedSystemPrompt.Content + compressionNote
	newHistory = append(newHistory, enhancedSystemPrompt)

	newHistory = append(newHistory, keptConversation...)
	newHistory = append(newHistory, history[len(history)-1])

	agent.Sessions.SetHistory(sessionKey, newHistory)
	agent.Sessions.Save(sessionKey)

	logger.WarnCF("agent", "Forced compression executed", map[string]any{
		"session_key":  sessionKey,
		"dropped_msgs": droppedCount,
		"new_count":    len(newHistory),
	})
}

// RebuildMessages rebuilds the LLM message list from current session state.
// Used after ForceCompression to get the updated context.
func (mm *MemoryManager) RebuildMessages(agent *AgentInstance, sessionKey, channel, chatID string) []providers.Message {
	newHistory := agent.Sessions.GetHistory(sessionKey)
	newSummary := agent.Sessions.GetSummary(sessionKey)
	return agent.ContextBuilder.BuildMessages(
		newHistory, newSummary, "", nil, channel, chatID,
	)
}
