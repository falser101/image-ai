package database

import (
	"github.com/image-ai/backend/config"
	"github.com/image-ai/backend/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Init(cfg *config.Config) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		panic("failed to open database: " + err.Error())
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Product{},
		&models.SellingPoint{},
		&models.Image{},
		&models.Gallery{},
		&models.StylePreset{},
		&models.ModelConfig{},
		&models.OssConfig{},
		&models.OperationLog{},
		&models.AITask{},
	); err != nil {
		panic("auto migrate failed: " + err.Error())
	}
	return db
}
