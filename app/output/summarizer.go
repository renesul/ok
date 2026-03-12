// OK - Lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OK contributors

package output

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"ok/internal/logger"
	"ok/providers"
)

// SummarizeAgent provides the agent configuration needed for summarization.
type SummarizeAgent interface {
	GetID() string
	GetModel() string
	GetContextWindow() int
	GetSummarizeMessageThreshold() int
	GetSummarizeTokenPercent() int
	GetProvider() providers.LLMProvider
}

// SessionAccess provides session history and summary operations.
type SessionAccess interface {
	GetHistory(key string) []providers.Message
	GetSummary(key string) string
	SetSummary(key, summary string)
	TruncateHistory(key string, keepLast int)
	Save(key string) error
}

// Summarizer manages conversation summarization.
type Summarizer struct {
	summarizing sync.Map
}

// NewSummarizer creates a new Summarizer.
func NewSummarizer() *Summarizer {
	return &Summarizer{}
}

// MaybeSummarize triggers summarization if the session history exceeds thresholds.
func (s *Summarizer) MaybeSummarize(agent SummarizeAgent, sessions SessionAccess, sessionKey string) {
	history := sessions.GetHistory(sessionKey)
	tokenEstimate := EstimateTokens(history)
	threshold := agent.GetContextWindow() * agent.GetSummarizeTokenPercent() / 100

	if len(history) > agent.GetSummarizeMessageThreshold() || tokenEstimate > threshold {
		summarizeKey := agent.GetID() + ":" + sessionKey
		if _, loading := s.summarizing.LoadOrStore(summarizeKey, true); !loading {
			go func() {
				defer s.summarizing.Delete(summarizeKey)
				logger.DebugC("agent", "Memory threshold reached. Optimizing conversation history...")
				s.SummarizeSession(agent, sessions, sessionKey)
			}()
		}
	}
}

// SummarizeSession summarizes the conversation history for a session.
func (s *Summarizer) SummarizeSession(agent SummarizeAgent, sessions SessionAccess, sessionKey string) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	history := sessions.GetHistory(sessionKey)
	summary := sessions.GetSummary(sessionKey)

	if len(history) <= 4 {
		return
	}

	toSummarize := history[:len(history)-4]

	maxMessageTokens := agent.GetContextWindow() / 2
	validMessages := make([]providers.Message, 0)
	omitted := false

	for _, m := range toSummarize {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		msgTokens := len(m.Content) / 2
		if msgTokens > maxMessageTokens {
			omitted = true
			continue
		}
		validMessages = append(validMessages, m)
	}

	if len(validMessages) == 0 {
		return
	}

	var finalSummary string
	if len(validMessages) > 10 {
		mid := len(validMessages) / 2
		part1 := validMessages[:mid]
		part2 := validMessages[mid:]

		s1, _ := SummarizeBatch(ctx, agent, part1, "")
		s2, _ := SummarizeBatch(ctx, agent, part2, "")

		mergePrompt := fmt.Sprintf(
			"Merge these two conversation summaries into one cohesive summary:\n\n1: %s\n\n2: %s",
			s1,
			s2,
		)
		resp, err := agent.GetProvider().Chat(
			ctx,
			[]providers.Message{{Role: "user", Content: mergePrompt}},
			nil,
			agent.GetModel(),
			map[string]any{
				"max_tokens":       1024,
				"temperature":      0.3,
				"prompt_cache_key": agent.GetID(),
			},
		)
		if err == nil {
			finalSummary = resp.Content
		} else {
			finalSummary = s1 + " " + s2
		}
	} else {
		finalSummary, _ = SummarizeBatch(ctx, agent, validMessages, summary)
	}

	if omitted && finalSummary != "" {
		finalSummary += "\n[Note: Some oversized messages were omitted from this summary for efficiency.]"
	}

	if finalSummary != "" {
		sessions.SetSummary(sessionKey, finalSummary)
		sessions.TruncateHistory(sessionKey, 4)
		sessions.Save(sessionKey)
	}
}

// SummarizeBatch summarizes a batch of messages.
func SummarizeBatch(
	ctx context.Context,
	agent SummarizeAgent,
	batch []providers.Message,
	existingSummary string,
) (string, error) {
	var sb strings.Builder
	sb.WriteString(
		"Provide a concise summary of this conversation segment, preserving core context and key points.\n",
	)
	if existingSummary != "" {
		sb.WriteString("Existing context: ")
		sb.WriteString(existingSummary)
		sb.WriteString("\n")
	}
	sb.WriteString("\nCONVERSATION:\n")
	for _, m := range batch {
		fmt.Fprintf(&sb, "%s: %s\n", m.Role, m.Content)
	}
	prompt := sb.String()

	response, err := agent.GetProvider().Chat(
		ctx,
		[]providers.Message{{Role: "user", Content: prompt}},
		nil,
		agent.GetModel(),
		map[string]any{
			"max_tokens":       1024,
			"temperature":      0.3,
			"prompt_cache_key": agent.GetID(),
		},
	)
	if err != nil {
		return "", err
	}
	return response.Content, nil
}

// EstimateTokens estimates the number of tokens in a message list.
// Uses a safe heuristic of 2.5 characters per token.
func EstimateTokens(messages []providers.Message) int {
	totalChars := 0
	for _, m := range messages {
		totalChars += utf8.RuneCountInString(m.Content)
	}
	return totalChars * 2 / 5
}
