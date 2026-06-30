package handlers

import (
	"strconv"

	"github.com/image-ai/backend/config"
	"github.com/image-ai/backend/models"
	"github.com/image-ai/backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ModelConfigHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewModelConfigHandler(db *gorm.DB, cfg *config.Config) *ModelConfigHandler {
	return &ModelConfigHandler{db: db, cfg: cfg}
}

func (h *ModelConfigHandler) List(c *gin.Context) {
	var list []models.ModelConfig
	h.db.Order("id desc").Find(&list)
	// 永远不在响应里回显明文 apiKey
	utils.OK(c, maskAPIKeys(list))
}

func (h *ModelConfigHandler) Create(c *gin.Context) {
	var m models.ModelConfig
	if err := c.ShouldBindJSON(&m); err != nil {
		utils.Fail(c, 400, "参数错误")
		return
	}
	// 内置 provider：自动补 baseUrl 和 name
	providers := providerCatalog()
	if p, ok := providers[m.Provider]; ok {
		if p.BuiltIn {
			if m.BaseURL == "" {
				m.BaseURL = p.BaseURL
			}
			if m.Name == "" {
				tag := m.Type
				if tag == "" {
					tag = "image"
				}
				m.Name = p.Label + " " + m.ModelName
			}
		}
	}
	if m.Name == "" {
		m.Name = m.Provider + " " + m.ModelName
	}
	if err := h.db.Create(&m).Error; err != nil {
		utils.Fail(c, 500, "创建失败")
		return
	}
	utils.OK(c, maskAPIKeys([]models.ModelConfig{m})[0])
}

// providerCatalog 暴露给同包内 handlers 复用（避免重复定义）
func providerCatalog() map[string]struct {
	BuiltIn bool
	BaseURL string
	Label   string
} {
	return map[string]struct {
		BuiltIn bool
		BaseURL string
		Label   string
	}{
		"minimax": {BuiltIn: true, BaseURL: "https://api.minimaxi.com", Label: "MiniMax（海螺）"},
	}
}

func (h *ModelConfigHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var m models.ModelConfig
	if err := h.db.First(&m, id).Error; err != nil {
		utils.Fail(c, 404, "不存在")
		return
	}
	// 用临时 struct 接收，保留零值
	var patch models.ModelConfig
	if err := c.ShouldBindJSON(&patch); err != nil {
		utils.Fail(c, 400, "参数错误")
		return
	}
	// 内置 provider：不允许把 baseUrl 改成非官方地址
	providers := providerCatalog()
	if p, ok := providers[m.Provider]; ok && p.BuiltIn {
		if patch.BaseURL == "" {
			patch.BaseURL = p.BaseURL
		} else if patch.BaseURL != p.BaseURL {
			// 强制还原为官方地址
			patch.BaseURL = p.BaseURL
		}
	}
	// apiKey 为空 → 视为"不改"，避免覆盖已有 key
	if patch.APIKey == "" {
		patch.APIKey = m.APIKey
	}
	h.db.Model(&m).Updates(patch)
	h.db.First(&m, id)
	utils.OK(c, maskAPIKeys([]models.ModelConfig{m})[0])
}

// maskAPIKeys 把列表里每条 model 的 apiKey 字段脱敏：
// 有值 → "****"（让前端知道有配置），空 → ""（让前端知道没配置）
// 真正的 key 永远不出现在响应里
func maskAPIKeys(list []models.ModelConfig) []models.ModelConfig {
	for i := range list {
		if list[i].APIKey != "" {
			list[i].APIKey = "****"
		}
	}
	return list
}

func (h *ModelConfigHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.db.Delete(&models.ModelConfig{}, id)
	utils.OK(c, nil)
}

// Presets 返回指定 provider+type 下的预置模型列表
func (h *ModelConfigHandler) Presets(c *gin.Context) {
	provider := c.Param("provider")
	t := c.Param("type")
	models := presetModelsFor(provider, t)
	utils.OK(c, models)
}

func presetModelsFor(provider, t string) []string {
	switch provider {
	case "minimax":
		if t == "image" {
			return []string{"image-01", "image-01-live"}
		}
		return []string{"abab6.5s-chat", "MiniMax-Text-01"}
	default:
		return []string{}
	}
}

type OssConfigHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewOssConfigHandler(db *gorm.DB, cfg *config.Config) *OssConfigHandler {
	return &OssConfigHandler{db: db, cfg: cfg}
}

func (h *OssConfigHandler) Get(c *gin.Context) {
	var m models.OssConfig
	if err := h.db.First(&m).Error; err != nil {
		// 单例尚未写入：默认 local，但仍然把实际的上传目录带上，
		// 前端展示「本地上传目录」用。
		utils.OK(c, models.OssConfig{Provider: "local", LocalDir: h.cfg.UploadDir})
		return
	}
	// local 模式下注入运行时上传目录（非 local provider 不返回 localDir）。
	if m.Provider == "" || m.Provider == "local" {
		m.LocalDir = h.cfg.UploadDir
	} else {
		m.LocalDir = ""
	}
	utils.OK(c, m)
}

func (h *OssConfigHandler) Update(c *gin.Context) {
	var m models.OssConfig
	if err := c.ShouldBindJSON(&m); err != nil {
		utils.Fail(c, 400, "参数错误")
		return
	}
	// 防止 provider 漏传：默认按 local 处理
	if m.Provider == "" {
		m.Provider = "local"
	}
	// local 模式下 OSS 字段清零，避免历史脏数据残留；非 local 模式保留用户输入。
	if m.Provider == "local" {
		m.Endpoint = ""
		m.Bucket = ""
		m.AccessKey = ""
		m.SecretKey = ""
		m.Region = ""
		m.Prefix = ""
		m.PublicHost = ""
		m.Enabled = false
	}
	var existing models.OssConfig
	if err := h.db.First(&existing).Error; err == nil {
		m.ID = existing.ID
		h.db.Save(&m)
	} else {
		h.db.Create(&m)
	}
	// 响应时再回填 localDir，保持和 Get 行为一致
	if m.Provider == "local" {
		m.LocalDir = h.cfg.UploadDir
	}
	utils.OK(c, m)
}

type StylePresetHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewStylePresetHandler(db *gorm.DB, cfg *config.Config) *StylePresetHandler {
	return &StylePresetHandler{db: db, cfg: cfg}
}

func (h *StylePresetHandler) List(c *gin.Context) {
	var list []models.StylePreset
	h.db.Order("id desc").Find(&list)
	utils.OK(c, list)
}

func (h *StylePresetHandler) Create(c *gin.Context) {
	var m models.StylePreset
	if err := c.ShouldBindJSON(&m); err != nil {
		utils.Fail(c, 400, "参数错误")
		return
	}
	h.db.Create(&m)
	utils.OK(c, m)
}

func (h *StylePresetHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var m models.StylePreset
	if err := h.db.First(&m, id).Error; err != nil {
		utils.Fail(c, 404, "不存在")
		return
	}
	var patch models.StylePreset
	if err := c.ShouldBindJSON(&patch); err != nil {
		utils.Fail(c, 400, "参数错误")
		return
	}
	h.db.Model(&m).Updates(patch)
	h.db.First(&m, id)
	utils.OK(c, m)
}

func (h *StylePresetHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.db.Delete(&models.StylePreset{}, id)
	utils.OK(c, nil)
}
