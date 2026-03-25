package domain

import (
	"context"
	"time"
)

type Conversation struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Source    string    `json:"source"`
	Channel   string    `json:"channel"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	ID             uint      `json:"id"`
	ConversationID uint      `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	SortOrder      int       `json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
}

type MessageEmbedding struct {
	MessageID      uint
	ConversationID uint
	Embedding      []float32
}

type ConversationRepository interface {
	Create(ctx context.Context, conversation *Conversation) error
	FindByID(ctx context.Context, id uint) (*Conversation, error)
	FindAll(ctx context.Context) ([]Conversation, error)
	Update(ctx context.Context, conversation *Conversation) error
	Delete(ctx context.Context, id uint) error
	Search(ctx context.Context, query string) ([]Conversation, error)
}

type MessageRepository interface {
	Create(ctx context.Context, message *Message) error
	CreateBatch(ctx context.Context, messages []Message) error
	FindByConversationID(ctx context.Context, conversationID uint) ([]Message, error)
	CountByConversationID(ctx context.Context, conversationID uint) (int64, error)
	IndexForSearch(ctx context.Context, messages []Message) error
	SaveEmbedding(ctx context.Context, messageID uint, conversationID uint, embedding []float32) error
	FindAllEmbeddings(ctx context.Context) ([]MessageEmbedding, error)
	DeleteByConversationID(ctx context.Context, conversationID uint) error
	DeleteSearchIndex(ctx context.Context, conversationID uint) error
	DeleteEmbeddings(ctx context.Context, conversationID uint) error
}
