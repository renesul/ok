package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

const maxContextMessages = 50

var ErrLLMNotConfigured = errors.New("llm not configured")

type ChatService struct {
	conversationRepository domain.ConversationRepository
	messageRepository      domain.MessageRepository
	embeddingService       *EmbeddingService
	agentService           *AgentService
	llmConfigured          bool
	log                    *zap.Logger
}

func NewChatService(
	conversationRepository domain.ConversationRepository,
	messageRepository domain.MessageRepository,
	embeddingService *EmbeddingService,
	agentService *AgentService,
	llmConfigured bool,
	log *zap.Logger,
) *ChatService {
	return &ChatService{
		conversationRepository: conversationRepository,
		messageRepository:      messageRepository,
		embeddingService:       embeddingService,
		agentService:           agentService,
		llmConfigured:          llmConfigured,
		log:                    log.Named("service.chat"),
	}
}

func (s *ChatService) CreateConversation(ctx context.Context, title string) (*domain.Conversation, error) {
	s.log.Debug("create conversation")

	if title == "" {
		title = "Nova conversa"
	}

	conversation := &domain.Conversation{
		Title:  title,
		Source:  "chat",
		Channel: "web",
	}

	if err := s.conversationRepository.Create(ctx, conversation); err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}
	return conversation, nil
}

func (s *ChatService) SendMessage(ctx context.Context, conversationID uint, content string, onEvent func(domain.AgentEvent)) (*domain.Message, error) {
	s.log.Debug("send message", zap.Uint("conversation_id", conversationID))

	if !s.llmConfigured {
		return nil, ErrLLMNotConfigured
	}

	conversation, err := s.conversationRepository.FindByID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("find conversation: %w", err)
	}
	if conversation == nil {
		return nil, ErrConversationNotFound
	}

	count, err := s.messageRepository.CountByConversationID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("count messages: %w", err)
	}

	userMessage := &domain.Message{
		ConversationID: conversationID,
		Role:           "user",
		Content:        content,
		SortOrder:      int(count),
	}
	if err := s.messageRepository.Create(ctx, userMessage); err != nil {
		return nil, fmt.Errorf("save user message: %w", err)
	}

	s.messageRepository.IndexForSearch(ctx, []domain.Message{*userMessage})
	if s.embeddingService != nil {
		s.embeddingService.EmbedAndStore(ctx, []domain.Message{*userMessage})
	}

	if count == 0 {
		title := content
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		conversation.Title = title
		conversation.UpdatedAt = time.Now()
		s.conversationRepository.Update(ctx, conversation)
	}

	contextMessages, contextErr := s.buildContext(ctx, conversationID)
	if contextErr != nil {
		s.log.Debug("build context failed", zap.Error(contextErr))
	}

	agentInput := content
	if len(contextMessages) > 1 {
		var parts []string
		for _, msg := range contextMessages[:len(contextMessages)-1] {
			parts = append(parts, msg.Role+": "+msg.Content)
		}
		agentInput = "Historico da conversa:\n" + strings.Join(parts, "\n") + "\n\nMensagem atual: " + content
	}

	var fullResponse strings.Builder
	s.agentService.RunStream(ctx, agentInput, func(event domain.AgentEvent) {
		switch event.Type {
		case "token":
			fullResponse.WriteString(event.Content)
		case "message":
			if fullResponse.Len() == 0 {
				fullResponse.WriteString(event.Content)
			}
		}
		if onEvent != nil {
			onEvent(event)
		}
	})

	assistantMessage := &domain.Message{
		ConversationID: conversationID,
		Role:           "assistant",
		Content:        fullResponse.String(),
		SortOrder:      int(count) + 1,
	}
	if err := s.messageRepository.Create(ctx, assistantMessage); err != nil {
		return nil, fmt.Errorf("save assistant message: %w", err)
	}

	s.messageRepository.IndexForSearch(ctx, []domain.Message{*assistantMessage})
	if s.embeddingService != nil {
		s.embeddingService.EmbedAndStore(ctx, []domain.Message{*assistantMessage})
	}

	conversation.UpdatedAt = time.Now()
	s.conversationRepository.Update(ctx, conversation)

	return assistantMessage, nil
}

func (s *ChatService) buildContext(ctx context.Context, conversationID uint) ([]domain.Message, error) {
	messages, err := s.messageRepository.FindByConversationID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	start := 0
	if len(messages) > maxContextMessages {
		start = len(messages) - maxContextMessages
	}

	var filtered []domain.Message
	for _, msg := range messages[start:] {
		if msg.Role == "user" || msg.Role == "assistant" {
			filtered = append(filtered, msg)
		}
	}
	return filtered, nil
}
