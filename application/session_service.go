package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

const sessionDuration = 7 * 24 * time.Hour

type SessionService struct {
	sessionRepository domain.SessionRepository
	log               *zap.Logger
}

func NewSessionService(sessionRepository domain.SessionRepository, log *zap.Logger) *SessionService {
	return &SessionService{
		sessionRepository: sessionRepository,
		log:               log.Named("service.session"),
	}
}

func (s *SessionService) CreateSession(ctx context.Context) (*domain.Session, error) {
	s.log.Debug("create session")

	session := &domain.Session{
		ID:        uuid.New().String(),
		ExpiresAt: time.Now().Add(sessionDuration),
	}

	if err := s.sessionRepository.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	s.log.Debug("session created", zap.String("session_id", session.ID))
	return session, nil
}

func (s *SessionService) ValidateSession(ctx context.Context, sessionID string) (bool, error) {
	s.log.Debug("validate session", zap.String("session_id", sessionID))

	session, err := s.sessionRepository.FindByID(ctx, sessionID)
	if err != nil {
		return false, fmt.Errorf("validate session: %w", err)
	}
	if session == nil {
		return false, nil
	}

	if time.Now().After(session.ExpiresAt) {
		s.log.Debug("session expired", zap.String("session_id", sessionID))
		s.sessionRepository.DeleteByID(ctx, sessionID)
		return false, nil
	}

	return true, nil
}

func (s *SessionService) DestroySession(ctx context.Context, sessionID string) error {
	s.log.Debug("destroy session", zap.String("session_id", sessionID))
	if err := s.sessionRepository.DeleteByID(ctx, sessionID); err != nil {
		return fmt.Errorf("destroy session: %w", err)
	}
	return nil
}
