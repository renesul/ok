package agent

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ConfigRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewConfigRepository(db *gorm.DB, log *zap.Logger) *ConfigRepository {
	return &ConfigRepository{db: db, log: log.Named("agent.config")}
}

func (r *ConfigRepository) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.WithContext(ctx).Raw("SELECT value FROM agent_config WHERE key = ?", key).Scan(&value).Error
	if err != nil {
		return "", fmt.Errorf("get config '%s': %w", key, err)
	}
	return value, nil
}

func (r *ConfigRepository) Set(ctx context.Context, key, value string) error {
	r.log.Debug("set config", zap.String("key", key))
	return r.db.WithContext(ctx).Exec(
		"INSERT OR REPLACE INTO agent_config (key, value) VALUES (?, ?)", key, value,
	).Error
}
