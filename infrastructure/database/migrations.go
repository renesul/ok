package database

import (
	"fmt"

	"github.com/renesul/ok/domain"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
