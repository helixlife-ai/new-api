package model

import (
	"fmt"

	"export-service/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB(cfg *config.Config) error {
	var err error
	DB, err = gorm.Open(postgres.Open(cfg.PostgresDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto migrate tables
	if err := DB.AutoMigrate(&FeishuTokenMapping{}, &ExportLog{}); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}
