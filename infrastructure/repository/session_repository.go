package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/renesul/ok/domain"
	"github.com/renesul/ok/infrastructure/database"
	"go.uber.org/zap"
)

type SessionRepository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewSessionRepository(db *sql.DB, log *zap.Logger) *SessionRepository {
	return &SessionRepository{db: db, log: log.Named("repository.session")}
}

func (r *SessionRepository) Create(ctx context.Context, session *domain.Session) error {
	ctx = database.Ctx(ctx)
	r.log.Debug("insert session", zap.String("id", session.ID))
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO sessions (id, expires_at, created_at) VALUES (?, ?, ?)",
		session.ID, session.ExpiresAt, session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (r *SessionRepository) FindByID(ctx context.Context, id string) (*domain.Session, error) {
	ctx = database.Ctx(ctx)
	r.log.Debug("find session by id", zap.String("id", id))
	var session domain.Session
	err := r.db.QueryRowContext(ctx,
		"SELECT id, expires_at, created_at FROM sessions WHERE id = ?", id,
	).Scan(&session.ID, &session.ExpiresAt, &session.CreatedAt)
	if err == sql.ErrNoRows {
		r.log.Debug("session not found", zap.String("id", id))
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find session by id: %w", err)
	}
	return &session, nil
}

func (r *SessionRepository) DeleteByID(ctx context.Context, id string) error {
	ctx = database.Ctx(ctx)
	r.log.Debug("delete session", zap.String("id", id))
	_, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (r *SessionRepository) DeleteExpired(ctx context.Context) error {
	ctx = database.Ctx(ctx)
	r.log.Debug("delete expired sessions")
	result, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < ?", time.Now())
	if err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	count, _ := result.RowsAffected()
	r.log.Debug("expired sessions deleted", zap.Int64("count", count))
	return nil
}
