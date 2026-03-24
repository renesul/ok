package repository

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MessageRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewMessageRepository(db *gorm.DB, log *zap.Logger) *MessageRepository {
	return &MessageRepository{db: db, log: log.Named("repository.message")}
}

func (r *MessageRepository) Create(ctx context.Context, message *domain.Message) error {
	r.log.Debug("insert message", zap.Uint("conversation_id", message.ConversationID))
	if err := r.db.WithContext(ctx).Create(message).Error; err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	return nil
}

func (r *MessageRepository) CreateBatch(ctx context.Context, messages []domain.Message) error {
	if len(messages) == 0 {
		return nil
	}
	r.log.Debug("insert messages batch", zap.Int("count", len(messages)))
	if err := r.db.WithContext(ctx).CreateInBatches(messages, 100).Error; err != nil {
		return fmt.Errorf("insert messages batch: %w", err)
	}
	return nil
}

func (r *MessageRepository) FindByConversationID(ctx context.Context, conversationID uint) ([]domain.Message, error) {
	r.log.Debug("find messages by conversation", zap.Uint("conversation_id", conversationID))
	var messages []domain.Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("sort_order ASC").
		Find(&messages).Error
	if err != nil {
		return nil, fmt.Errorf("find messages by conversation: %w", err)
	}
	r.log.Debug("messages found", zap.Int("count", len(messages)))
	return messages, nil
}

func (r *MessageRepository) CountByConversationID(ctx context.Context, conversationID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Message{}).Where("conversation_id = ?", conversationID).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count messages: %w", err)
	}
	return count, nil
}

func (r *MessageRepository) IndexForSearch(ctx context.Context, messages []domain.Message) error {
	if len(messages) == 0 {
		return nil
	}
	r.log.Debug("index messages for search", zap.Int("count", len(messages)))
	for _, msg := range messages {
		if msg.Content == "" {
			continue
		}
		err := r.db.WithContext(ctx).Exec(
			"INSERT INTO messages_fts(rowid, conversation_id, content) VALUES (?, ?, ?)",
			msg.ID, msg.ConversationID, msg.Content,
		).Error
		if err != nil {
			return fmt.Errorf("index message for search: %w", err)
		}
	}
	return nil
}

func (r *MessageRepository) SaveEmbedding(ctx context.Context, messageID uint, conversationID uint, embedding []float32) error {
	r.log.Debug("save embedding", zap.Uint("message_id", messageID))
	blob := float32ToBytes(embedding)
	return r.db.WithContext(ctx).Exec(
		"INSERT OR REPLACE INTO message_embeddings(message_id, conversation_id, embedding) VALUES (?, ?, ?)",
		messageID, conversationID, blob,
	).Error
}

func (r *MessageRepository) FindAllEmbeddings(ctx context.Context) ([]domain.MessageEmbedding, error) {
	r.log.Debug("find all embeddings")
	var rows []struct {
		MessageID      uint
		ConversationID uint
		Embedding      []byte
	}
	err := r.db.WithContext(ctx).
		Raw("SELECT message_id, conversation_id, embedding FROM message_embeddings").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("find embeddings: %w", err)
	}

	results := make([]domain.MessageEmbedding, len(rows))
	for i, row := range rows {
		results[i] = domain.MessageEmbedding{
			MessageID:      row.MessageID,
			ConversationID: row.ConversationID,
			Embedding:      bytesToFloat32(row.Embedding),
		}
	}
	return results, nil
}

func (r *MessageRepository) DeleteByConversationID(ctx context.Context, conversationID uint) error {
	r.log.Debug("delete messages by conversation", zap.Uint("conversation_id", conversationID))
	if err := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Delete(&domain.Message{}).Error; err != nil {
		return fmt.Errorf("delete messages by conversation: %w", err)
	}
	return nil
}

func (r *MessageRepository) DeleteSearchIndex(ctx context.Context, conversationID uint) error {
	r.log.Debug("delete search index by conversation", zap.Uint("conversation_id", conversationID))
	if err := r.db.WithContext(ctx).Exec("DELETE FROM messages_fts WHERE conversation_id = ?", conversationID).Error; err != nil {
		return fmt.Errorf("delete search index: %w", err)
	}
	return nil
}

func (r *MessageRepository) DeleteEmbeddings(ctx context.Context, conversationID uint) error {
	r.log.Debug("delete embeddings by conversation", zap.Uint("conversation_id", conversationID))
	if err := r.db.WithContext(ctx).Exec("DELETE FROM message_embeddings WHERE conversation_id = ?", conversationID).Error; err != nil {
		return fmt.Errorf("delete embeddings: %w", err)
	}
	return nil
}

func float32ToBytes(floats []float32) []byte {
	buf := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

func bytesToFloat32(data []byte) []float32 {
	floats := make([]float32, len(data)/4)
	for i := range floats {
		floats[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
	}
	return floats
}
