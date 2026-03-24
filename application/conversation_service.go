package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

var ErrConversationNotFound = errors.New("conversation not found")

type ConversationService struct {
	conversationRepository domain.ConversationRepository
	messageRepository      domain.MessageRepository
	embeddingService       *EmbeddingService
	log                    *zap.Logger
}

func NewConversationService(conversationRepository domain.ConversationRepository, messageRepository domain.MessageRepository, embeddingService *EmbeddingService, log *zap.Logger) *ConversationService {
	return &ConversationService{
		conversationRepository: conversationRepository,
		messageRepository:      messageRepository,
		embeddingService:       embeddingService,
		log:                    log.Named("service.conversation"),
	}
}

func (s *ConversationService) ListConversations(ctx context.Context) ([]domain.Conversation, error) {
	s.log.Debug("list conversations")
	conversations, err := s.conversationRepository.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	return conversations, nil
}

func (s *ConversationService) GetConversation(ctx context.Context, conversationID uint) (*domain.Conversation, error) {
	s.log.Debug("get conversation", zap.Uint("id", conversationID))
	conversation, err := s.conversationRepository.FindByID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}
	if conversation == nil {
		return nil, ErrConversationNotFound
	}
	return conversation, nil
}

func (s *ConversationService) GetMessages(ctx context.Context, conversationID uint) ([]domain.Message, error) {
	s.log.Debug("get messages", zap.Uint("conversation_id", conversationID))
	if _, err := s.GetConversation(ctx, conversationID); err != nil {
		return nil, err
	}
	messages, err := s.messageRepository.FindByConversationID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	return messages, nil
}

func (s *ConversationService) DeleteConversation(ctx context.Context, conversationID uint) error {
	s.log.Debug("delete conversation", zap.Uint("id", conversationID))
	if _, err := s.GetConversation(ctx, conversationID); err != nil {
		return err
	}
	if err := s.messageRepository.DeleteEmbeddings(ctx, conversationID); err != nil {
		s.log.Debug("delete embeddings failed", zap.Error(err))
	}
	if err := s.messageRepository.DeleteSearchIndex(ctx, conversationID); err != nil {
		s.log.Debug("delete search index failed", zap.Error(err))
	}
	if err := s.messageRepository.DeleteByConversationID(ctx, conversationID); err != nil {
		return fmt.Errorf("delete messages: %w", err)
	}
	if err := s.conversationRepository.Delete(ctx, conversationID); err != nil {
		return fmt.Errorf("delete conversation: %w", err)
	}
	return nil
}

func (s *ConversationService) SearchConversations(ctx context.Context, query string) ([]domain.Conversation, error) {
	s.log.Debug("search conversations", zap.String("query", query))

	if s.embeddingService != nil && s.embeddingService.Enabled() {
		conversations, err := s.embeddingService.SearchSemantic(ctx, query)
		if err == nil && len(conversations) > 0 {
			return conversations, nil
		}
		s.log.Debug("semantic search fallback to fts5", zap.Error(err))
	}

	conversations, err := s.conversationRepository.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("search conversations: %w", err)
	}
	return conversations, nil
}
