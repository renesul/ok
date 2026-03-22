package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

var ErrNotFound = errors.New("not found")
var ErrValidation = errors.New("validation failed")

type UserService struct {
	repository domain.UserRepository
	validate   *validator.Validate
	log        *zap.Logger
}

func NewUserService(repository domain.UserRepository, log *zap.Logger) *UserService {
	return &UserService{
		repository: repository,
		validate:   validator.New(),
		log:        log.Named("service.user"),
	}
}

func (s *UserService) CreateUser(ctx context.Context, user *domain.User) error {
	s.log.Debug("create user", zap.String("name", user.Name), zap.String("email", user.Email))
	if err := s.validate.Struct(user); err != nil {
		s.log.Debug("validation failed", zap.Error(err))
		return fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	if err := s.repository.Create(ctx, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	s.log.Debug("user created", zap.Uint("id", user.ID))
	return nil
}

func (s *UserService) GetUser(ctx context.Context, id uint) (*domain.User, error) {
	s.log.Debug("get user", zap.Uint("id", id))
	user, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		s.log.Debug("user not found", zap.Uint("id", id))
		return nil, ErrNotFound
	}
	return user, nil
}

func (s *UserService) ListUsers(ctx context.Context) ([]domain.User, error) {
	s.log.Debug("list users")
	users, err := s.repository.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	s.log.Debug("users listed", zap.Int("count", len(users)))
	return users, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id uint, name, email string) (*domain.User, error) {
	s.log.Debug("update user", zap.Uint("id", id), zap.String("name", name), zap.String("email", email))
	user, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	if user == nil {
		return nil, ErrNotFound
	}

	user.Name = name
	user.Email = email

	if err := s.validate.Struct(user); err != nil {
		s.log.Debug("validation failed", zap.Error(err))
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	if err := s.repository.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return user, nil
}

func (s *UserService) DeleteUser(ctx context.Context, id uint) error {
	s.log.Debug("delete user", zap.Uint("id", id))
	user, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if user == nil {
		return ErrNotFound
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	s.log.Debug("user deleted", zap.Uint("id", id))
	return nil
}
