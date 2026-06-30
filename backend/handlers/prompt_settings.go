package handlers

import (
	"github.com/image-ai/backend/config"
	"github.com/image-ai/backend/models"
	"github.com/image-ai/backend/services"
	"github.com/image-ai/backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PromptSettingsHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewPromptSettingsHandler(db *gorm.DB, cfg *config.Config) *PromptSettingsHandler {
	return &PromptSettingsHandler{db: db, cfg: cfg}
}

// Get 读取当前 system prompt。idempotent：行不存在时返回默认值。
func (h *PromptSettingsHandler) Get(c *gin.Context) {
	var s models.PromptSettings
	if err := h.db.First(&s, 1).Error; err == nil && s.SystemInstruction != "" {
		utils.OK(c, s)
		return
	}
	utils.OK(c, gin.H{
		"id":                1,
		"systemInstruction": services.DefaultSystemInstruction,
		"updatedBy":         0,
	})
}

// Update 管理员保存新的 system prompt。空字符串拒绝。
func (h *PromptSettingsHandler) Update(c *gin.Context) {
	uid, _ := c.Get("userId")
	var req struct {
		SystemInstruction string `json:"systemInstruction" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.SystemInstruction == "" {
		utils.Fail(c, 400, "提示词不能为空")
		return
	}
	uidVal, _ := uid.(uint)
	s := models.PromptSettings{
		ID:                1,
		SystemInstruction: req.SystemInstruction,
		UpdatedBy:         uidVal,
	}
	// upsert：先尝试 update，不存在则 create
	res := h.db.Model(&models.PromptSettings{}).Where("id = ?", 1).Updates(map[string]any{
		"system_instruction": req.SystemInstruction,
		"updated_by":         uidVal,
	})
	if res.Error != nil {
		utils.Fail(c, 500, "保存失败")
		return
	}
	if res.RowsAffected == 0 {
		if err := h.db.Create(&s).Error; err != nil {
			utils.Fail(c, 500, "保存失败")
			return
		}
	}
	// 读回最新值返回
	var fresh models.PromptSettings
	h.db.First(&fresh, 1)
	utils.OK(c, fresh)
}

// Reset 管理员一键恢复默认提示词。
func (h *PromptSettingsHandler) Reset(c *gin.Context) {
	uid, _ := c.Get("userId")
	uidVal, _ := uid.(uint)
	res := h.db.Model(&models.PromptSettings{}).Where("id = ?", 1).Updates(map[string]any{
		"system_instruction": services.DefaultSystemInstruction,
		"updated_by":         uidVal,
	})
	if res.Error != nil {
		utils.Fail(c, 500, "重置失败")
		return
	}
	if res.RowsAffected == 0 {
		if err := h.db.Create(&models.PromptSettings{
			ID:                1,
			SystemInstruction: services.DefaultSystemInstruction,
			UpdatedBy:         uidVal,
		}).Error; err != nil {
			utils.Fail(c, 500, "重置失败")
			return
		}
	}
	utils.OK(c, gin.H{"systemInstruction": services.DefaultSystemInstruction, "isDefault": true})
}
