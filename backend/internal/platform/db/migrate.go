package db

import (
	"github.com/srujanpendurthi/transcriptflow/internal/platform/models"
	"gorm.io/gorm"
)

// RunMigrations auto-migrates all models in dependency order.
func RunMigrations(database *gorm.DB) error {
	return database.AutoMigrate(
		&models.User{},
		&models.APIKey{},
		&models.UsageLog{},
		&models.TranscriptionJob{},
	)
}
