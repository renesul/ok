package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewUserRepository(db *gorm.DB, log *zap.Logger) *UserRepository {
	return &UserRepository{db: db, log: log.Named("repository.user")}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	r.log.Debug("insert user", zap.String("name", user.Name), zap.String("email", user.Email))
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	r.log.Debug("user inserted", zap.Uint("id", user.ID))
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	r.log.Debug("find user by id", zap.Uint("id", id))
	var user domain.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		r.log.Debug("user not found", zap.Uint("id", id))
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) FindAll(ctx context.Context) ([]domain.User, error) {
	r.log.Debug("find all users")
	var users []domain.User
	if err := r.db.WithContext(ctx).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("find all users: %w", err)
	}
	r.log.Debug("users found", zap.Int("count", len(users)))
	return users, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	r.log.Debug("update user", zap.Uint("id", user.ID), zap.String("name", user.Name), zap.String("email", user.Email))
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	r.log.Debug("delete user", zap.Uint("id", id))
	result := r.db.WithContext(ctx).Delete(&domain.User{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete user: %w", result.Error)
	}
	r.log.Debug("user deleted", zap.Int64("rows_affected", result.RowsAffected))
	return nil
}
