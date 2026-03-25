package agent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/renesul/ok/infrastructure/database"
	"go.uber.org/zap"
)

type ConfigRepository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewConfigRepository(db *sql.DB, log *zap.Logger) *ConfigRepository {
	return &ConfigRepository{db: db, log: log.Named("agent.config")}
}

func (r *ConfigRepository) Get(ctx context.Context, key string) (string, error) {
	ctx = database.Ctx(ctx)
	var value string
	err := r.db.QueryRowContext(ctx, "SELECT value FROM agent_config WHERE key = ?", key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get config '%s': %w", key, err)
	}
	return value, nil
}

func (r *ConfigRepository) Set(ctx context.Context, key, value string) error {
	ctx = database.Ctx(ctx)
	r.log.Debug("set config", zap.String("key", key))
	_, err := r.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO agent_config (key, value) VALUES (?, ?)", key, value,
	)
	if err != nil {
		return fmt.Errorf("set config '%s': %w", key, err)
	}
	return nil
}
