package services

import (
	"errors"

	"github.com/image-ai/backend/config"
	"github.com/image-ai/backend/models"
	"github.com/image-ai/backend/utils"

	"gorm.io/gorm"
)

func EnsureDefaultAdmin(db *gorm.DB, cfg *config.Config) error {
	var count int64
	if err := db.Model(&models.User{}).Where("username = ?", cfg.DefaultAdmin.Username).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := utils.HashPassword(cfg.DefaultAdmin.Password)
	if err != nil {
		return err
	}
	admin := models.User{
		Username: cfg.DefaultAdmin.Username,
		Password: hash,
		Nickname: "系统管理员",
		Role:     "admin",
		Status:   "active",
	}
	return db.Create(&admin).Error
}

func EnsureUser(db *gorm.DB, username, password, nickname, role string) (*models.User, error) {
	if role != "admin" && role != "employee" {
		return nil, errors.New("invalid role")
	}
	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}
	u := &models.User{
		Username: username,
		Password: hash,
		Nickname: nickname,
		Role:     role,
		Status:   "active",
	}
	if err := db.Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}
