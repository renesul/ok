package domain

import (
	"context"
	"time"
)

type Session struct {
	ID        string    `json:"-" gorm:"primaryKey;type:text"`
	ExpiresAt time.Time `json:"-"`
	CreatedAt time.Time `json:"-"`
}

type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	FindByID(ctx context.Context, id string) (*Session, error)
	DeleteByID(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) error
}
