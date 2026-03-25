package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/renesul/ok/domain"
	"github.com/renesul/ok/infrastructure/database"
	"go.uber.org/zap"
)

type ConversationRepository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewConversationRepository(db *sql.DB, log *zap.Logger) *ConversationRepository {
	return &ConversationRepository{db: db, log: log.Named("repository.conversation")}
}

func (r *ConversationRepository) Create(ctx context.Context, conversation *domain.Conversation) error {
	ctx = database.Ctx(ctx)
	r.log.Debug("insert conversation", zap.String("title", conversation.Title))
	now := time.Now()
	if conversation.CreatedAt.IsZero() {
		conversation.CreatedAt = now
	}
	if conversation.UpdatedAt.IsZero() {
		conversation.UpdatedAt = now
	}
	if conversation.Source == "" {
		conversation.Source = "import"
	}
	if conversation.Channel == "" {
		conversation.Channel = "web"
	}

	result, err := r.db.ExecContext(ctx,
		"INSERT INTO conversations (title, source, channel, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		conversation.Title, conversation.Source, conversation.Channel, conversation.CreatedAt, conversation.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert conversation: %w", err)
	}
	id, _ := result.LastInsertId()
	conversation.ID = uint(id)
	r.log.Debug("conversation inserted", zap.Uint("id", conversation.ID))
	return nil
}

func (r *ConversationRepository) FindByID(ctx context.Context, id uint) (*domain.Conversation, error) {
	ctx = database.Ctx(ctx)
	r.log.Debug("find conversation by id", zap.Uint("id", id))
	var c domain.Conversation
	err := r.db.QueryRowContext(ctx,
		"SELECT id, title, source, channel, created_at, updated_at FROM conversations WHERE id = ?", id,
	).Scan(&c.ID, &c.Title, &c.Source, &c.Channel, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		r.log.Debug("conversation not found", zap.Uint("id", id))
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find conversation by id: %w", err)
	}
	return &c, nil
}

func (r *ConversationRepository) FindAll(ctx context.Context) ([]domain.Conversation, error) {
	ctx = database.Ctx(ctx)
	r.log.Debug("find all conversations")
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, title, source, channel, created_at, updated_at FROM conversations ORDER BY updated_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("find all conversations: %w", err)
	}
	defer rows.Close()

	var conversations []domain.Conversation
	for rows.Next() {
		var c domain.Conversation
		if err := rows.Scan(&c.ID, &c.Title, &c.Source, &c.Channel, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan conversation row: %w", err)
		}
		conversations = append(conversations, c)
	}
	r.log.Debug("conversations found", zap.Int("count", len(conversations)))
	return conversations, rows.Err()
}

func (r *ConversationRepository) Update(ctx context.Context, conversation *domain.Conversation) error {
	ctx = database.Ctx(ctx)
	r.log.Debug("update conversation", zap.Uint("id", conversation.ID))
	_, err := r.db.ExecContext(ctx,
		"UPDATE conversations SET title=?, source=?, channel=?, updated_at=? WHERE id=?",
		conversation.Title, conversation.Source, conversation.Channel, conversation.UpdatedAt, conversation.ID,
	)
	if err != nil {
		return fmt.Errorf("update conversation: %w", err)
	}
	return nil
}

func (r *ConversationRepository) Delete(ctx context.Context, id uint) error {
	ctx = database.Ctx(ctx)
	r.log.Debug("delete conversation", zap.Uint("id", id))
	_, err := r.db.ExecContext(ctx, "DELETE FROM conversations WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete conversation: %w", err)
	}
	return nil
}

func (r *ConversationRepository) Search(ctx context.Context, query string) ([]domain.Conversation, error) {
	ctx = database.Ctx(ctx)
	r.log.Debug("search conversations", zap.String("query", query))

	// Step 1: FTS5 search for matching conversation IDs
	ftsRows, err := r.db.QueryContext(ctx,
		"SELECT DISTINCT conversation_id FROM messages_fts WHERE messages_fts MATCH ?", query,
	)
	if err != nil {
		return nil, fmt.Errorf("search conversations: %w", err)
	}
	defer ftsRows.Close()

	var ids []uint
	for ftsRows.Next() {
		var id uint
		if err := ftsRows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan fts row: %w", err)
		}
		ids = append(ids, id)
	}
	if err := ftsRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fts rows: %w", err)
	}

	if len(ids) == 0 {
		return []domain.Conversation{}, nil
	}

	// Step 2: Load conversations by IDs
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx,
		"SELECT id, title, source, channel, created_at, updated_at FROM conversations WHERE id IN ("+placeholders+") ORDER BY updated_at DESC",
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("search conversations load: %w", err)
	}
	defer rows.Close()

	var conversations []domain.Conversation
	for rows.Next() {
		var c domain.Conversation
		if err := rows.Scan(&c.ID, &c.Title, &c.Source, &c.Channel, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan conversation row: %w", err)
		}
		conversations = append(conversations, c)
	}

	r.log.Debug("search results", zap.Int("count", len(conversations)))
	return conversations, rows.Err()
}
