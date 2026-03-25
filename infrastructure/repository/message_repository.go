package repository

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/renesul/ok/domain"
	"github.com/renesul/ok/infrastructure/database"
	"go.uber.org/zap"
)

type MessageRepository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewMessageRepository(db *sql.DB, log *zap.Logger) *MessageRepository {
	return &MessageRepository{db: db, log: log.Named("repository.message")}
}

func (r *MessageRepository) Create(ctx context.Context, message *domain.Message) error {
	ctx = database.Ctx(ctx)
	r.log.Debug("insert message", zap.Uint("conversation_id", message.ConversationID))
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}
	result, err := r.db.ExecContext(ctx,
		"INSERT INTO messages (conversation_id, role, content, sort_order, created_at) VALUES (?, ?, ?, ?, ?)",
		message.ConversationID, message.Role, message.Content, message.SortOrder, message.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	id, _ := result.LastInsertId()
	message.ID = uint(id)
	return nil
}

func (r *MessageRepository) CreateBatch(ctx context.Context, messages []domain.Message) error {
	ctx = database.Ctx(ctx)
	if len(messages) == 0 {
		return nil
	}
	r.log.Debug("insert messages batch", zap.Int("count", len(messages)))

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin batch tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		"INSERT INTO messages (conversation_id, role, content, sort_order, created_at) VALUES (?, ?, ?, ?, ?)",
	)
	if err != nil {
		return fmt.Errorf("prepare batch insert: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for i := range messages {
		if messages[i].CreatedAt.IsZero() {
			messages[i].CreatedAt = now
		}
		result, err := stmt.ExecContext(ctx,
			messages[i].ConversationID, messages[i].Role, messages[i].Content, messages[i].SortOrder, messages[i].CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert message batch item %d: %w", i, err)
		}
		id, _ := result.LastInsertId()
		messages[i].ID = uint(id)
	}

	return tx.Commit()
}

func (r *MessageRepository) FindByConversationID(ctx context.Context, conversationID uint) ([]domain.Message, error) {
	ctx = database.Ctx(ctx)
	r.log.Debug("find messages by conversation", zap.Uint("conversation_id", conversationID))
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, conversation_id, role, content, sort_order, created_at FROM messages WHERE conversation_id = ? ORDER BY sort_order ASC",
		conversationID,
	)
	if err != nil {
		return nil, fmt.Errorf("find messages by conversation: %w", err)
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		var m domain.Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.SortOrder, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message row: %w", err)
		}
		messages = append(messages, m)
	}
	r.log.Debug("messages found", zap.Int("count", len(messages)))
	return messages, rows.Err()
}

func (r *MessageRepository) CountByConversationID(ctx context.Context, conversationID uint) (int64, error) {
	ctx = database.Ctx(ctx)
	var count int64
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM messages WHERE conversation_id = ?", conversationID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count messages: %w", err)
	}
	return count, nil
}

func (r *MessageRepository) IndexForSearch(ctx context.Context, messages []domain.Message) error {
	ctx = database.Ctx(ctx)
	if len(messages) == 0 {
		return nil
	}
	r.log.Debug("index messages for search", zap.Int("count", len(messages)))
	for _, msg := range messages {
		if msg.Content == "" {
			continue
		}
		_, err := r.db.ExecContext(ctx,
			"INSERT INTO messages_fts(rowid, conversation_id, content) VALUES (?, ?, ?)",
			msg.ID, msg.ConversationID, msg.Content,
		)
		if err != nil {
			return fmt.Errorf("index message for search: %w", err)
		}
	}
	return nil
}

func (r *MessageRepository) SaveEmbedding(ctx context.Context, messageID uint, conversationID uint, embedding []float32) error {
	ctx = database.Ctx(ctx)
	r.log.Debug("save embedding", zap.Uint("message_id", messageID))
	blob := float32ToBytes(embedding)
	_, err := r.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO message_embeddings(message_id, conversation_id, embedding) VALUES (?, ?, ?)",
		messageID, conversationID, blob,
	)
	if err != nil {
		return fmt.Errorf("save embedding: %w", err)
	}
	return nil
}

func (r *MessageRepository) FindAllEmbeddings(ctx context.Context) ([]domain.MessageEmbedding, error) {
	ctx = database.Ctx(ctx)
	r.log.Debug("find all embeddings")
	rows, err := r.db.QueryContext(ctx,
		"SELECT message_id, conversation_id, embedding FROM message_embeddings",
	)
	if err != nil {
		return nil, fmt.Errorf("find embeddings: %w", err)
	}
	defer rows.Close()

	var results []domain.MessageEmbedding
	for rows.Next() {
		var messageID, conversationID uint
		var blob []byte
		if err := rows.Scan(&messageID, &conversationID, &blob); err != nil {
			return nil, fmt.Errorf("scan embedding row: %w", err)
		}
		results = append(results, domain.MessageEmbedding{
			MessageID:      messageID,
			ConversationID: conversationID,
			Embedding:      bytesToFloat32(blob),
		})
	}
	return results, rows.Err()
}

func (r *MessageRepository) DeleteByConversationID(ctx context.Context, conversationID uint) error {
	ctx = database.Ctx(ctx)
	r.log.Debug("delete messages by conversation", zap.Uint("conversation_id", conversationID))
	_, err := r.db.ExecContext(ctx, "DELETE FROM messages WHERE conversation_id = ?", conversationID)
	if err != nil {
		return fmt.Errorf("delete messages by conversation: %w", err)
	}
	return nil
}

func (r *MessageRepository) DeleteSearchIndex(ctx context.Context, conversationID uint) error {
	ctx = database.Ctx(ctx)
	r.log.Debug("delete search index by conversation", zap.Uint("conversation_id", conversationID))
	_, err := r.db.ExecContext(ctx, "DELETE FROM messages_fts WHERE conversation_id = ?", conversationID)
	if err != nil {
		return fmt.Errorf("delete search index: %w", err)
	}
	return nil
}

func (r *MessageRepository) DeleteEmbeddings(ctx context.Context, conversationID uint) error {
	ctx = database.Ctx(ctx)
	r.log.Debug("delete embeddings by conversation", zap.Uint("conversation_id", conversationID))
	_, err := r.db.ExecContext(ctx, "DELETE FROM message_embeddings WHERE conversation_id = ?", conversationID)
	if err != nil {
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
