package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SessionRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewSessionRepository(db *gorm.DB, log *zap.Logger) *SessionRepository {
	return &SessionRepository{db: db, log: log.Named("repository.session")}
}

func (r *SessionRepository) Create(ctx context.Context, session *domain.Session) error {
	r.log.Debug("insert session", zap.String("id", session.ID))
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (r *SessionRepository) FindByID(ctx context.Context, id string) (*domain.Session, error) {
	r.log.Debug("find session by id", zap.String("id", id))
	var session domain.Session
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		r.log.Debug("session not found", zap.String("id", id))
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find session by id: %w", err)
	}
	return &session, nil
}

func (r *SessionRepository) DeleteByID(ctx context.Context, id string) error {
	r.log.Debug("delete session", zap.String("id", id))
	if err := r.db.WithContext(ctx).Delete(&domain.Session{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (r *SessionRepository) DeleteExpired(ctx context.Context) error {
	r.log.Debug("delete expired sessions")
	result := r.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&domain.Session{})
	if result.Error != nil {
		return fmt.Errorf("delete expired sessions: %w", result.Error)
	}
	r.log.Debug("expired sessions deleted", zap.Int64("count", result.RowsAffected))
	return nil
}
