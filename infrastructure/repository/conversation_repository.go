package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ConversationRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewConversationRepository(db *gorm.DB, log *zap.Logger) *ConversationRepository {
	return &ConversationRepository{db: db, log: log.Named("repository.conversation")}
}

func (r *ConversationRepository) Create(ctx context.Context, conversation *domain.Conversation) error {
	r.log.Debug("insert conversation", zap.String("title", conversation.Title))
	if err := r.db.WithContext(ctx).Create(conversation).Error; err != nil {
		return fmt.Errorf("insert conversation: %w", err)
	}
	r.log.Debug("conversation inserted", zap.Uint("id", conversation.ID))
	return nil
}

func (r *ConversationRepository) FindByID(ctx context.Context, id uint) (*domain.Conversation, error) {
	r.log.Debug("find conversation by id", zap.Uint("id", id))
	var conversation domain.Conversation
	err := r.db.WithContext(ctx).First(&conversation, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		r.log.Debug("conversation not found", zap.Uint("id", id))
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find conversation by id: %w", err)
	}
	return &conversation, nil
}

func (r *ConversationRepository) FindAll(ctx context.Context) ([]domain.Conversation, error) {
	r.log.Debug("find all conversations")
	var conversations []domain.Conversation
	err := r.db.WithContext(ctx).
		Order("updated_at DESC").
		Find(&conversations).Error
	if err != nil {
		return nil, fmt.Errorf("find all conversations: %w", err)
	}
	r.log.Debug("conversations found", zap.Int("count", len(conversations)))
	return conversations, nil
}

func (r *ConversationRepository) Update(ctx context.Context, conversation *domain.Conversation) error {
	r.log.Debug("update conversation", zap.Uint("id", conversation.ID))
	if err := r.db.WithContext(ctx).Save(conversation).Error; err != nil {
		return fmt.Errorf("update conversation: %w", err)
	}
	return nil
}

func (r *ConversationRepository) Delete(ctx context.Context, id uint) error {
	r.log.Debug("delete conversation", zap.Uint("id", id))
	if err := r.db.WithContext(ctx).Delete(&domain.Conversation{}, id).Error; err != nil {
		return fmt.Errorf("delete conversation: %w", err)
	}
	return nil
}

func (r *ConversationRepository) Search(ctx context.Context, query string) ([]domain.Conversation, error) {
	r.log.Debug("search conversations", zap.String("query", query))

	var conversationIDs []uint
	err := r.db.WithContext(ctx).
		Raw("SELECT DISTINCT conversation_id FROM messages_fts WHERE messages_fts MATCH ?", query).
		Scan(&conversationIDs).Error
	if err != nil {
		return nil, fmt.Errorf("search conversations: %w", err)
	}

	if len(conversationIDs) == 0 {
		return []domain.Conversation{}, nil
	}

	var conversations []domain.Conversation
	err = r.db.WithContext(ctx).
		Where("id IN ?", conversationIDs).
		Order("updated_at DESC").
		Find(&conversations).Error
	if err != nil {
		return nil, fmt.Errorf("search conversations load: %w", err)
	}

	r.log.Debug("search results", zap.Int("count", len(conversations)))
	return conversations, nil
}
