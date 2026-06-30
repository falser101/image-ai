package handlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/image-ai/backend/config"
	"github.com/image-ai/backend/models"
	"github.com/image-ai/backend/services"
	"github.com/image-ai/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AIHandler struct {
	db  *gorm.DB
	cfg *config.Config
	ai  *services.AIService
}

func NewAIHandler(db *gorm.DB, cfg *config.Config) *AIHandler {
	return &AIHandler{db: db, cfg: cfg, ai: services.NewAIService(db, cfg)}
}

// Analyze 上传图片后做视觉分析。**必须**先选产品（productId 必填），
// 不再自动创建产品；上传图归入选定产品，更新产品元信息并追加新卖点。
func (h *AIHandler) Analyze(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")

	// 1) productId 必填
	pidStr := c.PostForm("productId")
	if pidStr == "" {
		utils.Fail(c, 400, "请先选择产品（先去「产品」页创建一个）")
		return
	}
	pid := parseUint(pidStr)
	if pid == 0 {
		utils.Fail(c, 400, "产品 ID 无效")
		return
	}
	var product models.Product
	pq := h.db.Model(&models.Product{}).Where("id = ?", pid)
	if role != "admin" {
		pq = pq.Where("user_id = ?", uid)
	}
	if err := pq.First(&product).Error; err != nil {
		utils.Fail(c, 404, "产品不存在或无权访问")
		return
	}

	// 2) 可选：指定用哪个视觉模型配置
	var modelCfgID *uint
	if s := c.PostForm("modelConfigId"); s != "" {
		v := parseUint(s)
		if v > 0 {
			modelCfgID = &v
		}
	}

	// 3) 上传原图（在调视觉模型前落盘，失败也能看到图）
	img, err := uploadImage(c, h.db, h.cfg, &pid)
	if err != nil {
		utils.Fail(c, 400, "上传失败: "+err.Error())
		return
	}
	// 图片直接绑定到选定产品
	img.ProductID = &pid

	taskID := uuid.NewString()
	task := models.AITask{
		ID:      taskID,
		UserID:  uid.(uint),
		Type:    "analyze",
		Status:  "running",
		Payload: fmt.Sprintf(`{"imageId":%d,"productId":%d,"modelConfigId":%d}`, img.ID, pid, modelCfgIDOrZero(modelCfgID)),
	}
	h.db.Create(&task)

	res, usedCfg, err := h.ai.Analyze(c.Request.Context(), uid.(uint), img.ID, modelCfgID, product.Name)
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		h.db.Save(&task)
		utils.Fail(c, 500, "解析失败: "+err.Error())
		return
	}
	// 4) 写回 image 元数据
	jsonSP, _ := json.Marshal(res.SellingPoints)
	img.Prompt = res.Prompt
	img.SellingPts = string(jsonSP)
	img.Analyzed = true
	h.db.Save(&img)

	// 5) 更新产品的反规范化字段 + 把封面图换成最新这张
	product.Prompt = res.Prompt
	product.SellingPts = string(jsonSP)
	product.ImageID = &img.ID
	h.db.Save(&product)

	// 6) 追加本次的 AI 卖点（保留已有的人工/AI 卖点）
	for _, p := range res.SellingPoints {
		h.db.Create(&models.SellingPoint{
			UserID:    uid.(uint),
			ProductID: &pid,
			Content:   p,
			Source:    "ai",
		})
	}

	task.Status = "success"
	task.Result = fmt.Sprintf(`{"imageId":%d,"productId":%d,"sellingPoints":%s,"prompt":%q,"modelName":%q}`,
		img.ID, pid, string(jsonSP), res.Prompt, modelNameOf(usedCfg))
	task.Progress = 100
	h.db.Save(&task)
	// AI 调用本身单独写一条 operation_logs（不走中间件），填上 token 消耗
	h.recordAILog(c, uid.(uint), productNameOf(product), "ai.analyze", usedCfg, res.TokenUsage)
	utils.OK(c, gin.H{
		"taskId":        taskID,
		"imageId":       img.ID,
		"productId":     pid,
		"sellingPoints": res.SellingPoints,
		"prompt":        res.Prompt,
		"imageUrl":      services.BuildImageURL(h.cfg.UploadDir, img.Path),
		"modelName":     modelNameOf(usedCfg),
	})
}

// productNameOf 安全读 product name
func productNameOf(p models.Product) string {
	if p.Name == "" {
		return fmt.Sprintf("#%d", p.ID)
	}
	return p.Name
}

// recordAILog 落一条 AI 调用日志到 operation_logs。
// 失败也不应影响主业务流程，所以忽略 DB 错误。
func (h *AIHandler) recordAILog(c *gin.Context, uid uint, target, action string, cfg *models.ModelConfig, usage services.TokenUsage) {
	uname, _ := c.Get("username")
	modelName := ""
	if cfg != nil {
		modelName = cfg.Name
	}
	detail := "调用视觉/生图模型「" + modelName + "」"
	if target != "" {
		detail += " · 目标：" + target
	}
	if usage.Total > 0 {
		detail += fmt.Sprintf(" · token: prompt=%d completion=%d total=%d", usage.Prompt, usage.Completion, usage.Total)
	}
	resourceID := ""
	if cfg != nil {
		resourceID = modelName
	}
	h.db.Create(&models.OperationLog{
		UserID:           uid,
		Username:         asStringUsername(uname),
		Action:           action,
		Resource:         "model",
		ResourceID:       resourceID,
		Detail:           detail,
		IP:               c.ClientIP(),
		Tokens:           usage.Total,
		TokensPrompt:     usage.Prompt,
		TokensCompletion: usage.Completion,
	})
}

func asStringUsername(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// modelNameOf 安全拿 ModelConfig 名字，避免 nil 解引用
func modelNameOf(c *models.ModelConfig) string {
	if c == nil {
		return ""
	}
	return c.Name
}

// modelCfgIDOrZero 把 *uint 安全转成可放进 JSON 的整数（nil → 0）
func modelCfgIDOrZero(p *uint) uint {
	if p == nil {
		return 0
	}
	return *p
}

type generateReq struct {
	ProductID      *uint  `json:"productId"`
	SourceImageID  *uint  `json:"sourceImageId"`
	UseAsSubject   bool   `json:"useAsSubject"`
	ModelConfigID  *uint  `json:"modelConfigId"`
	StyleID        *uint  `json:"styleId"`
	Prompt         string `json:"prompt"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	PromptOptimizer bool  `json:"promptOptimizer"`
}

func (h *AIHandler) Generate(c *gin.Context) {
	uid, _ := c.Get("userId")
	var req generateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, 400, "参数错误")
		return
	}
	// 没传 prompt 时从 source image 的存储 prompt 拿（产品图场景）；仍空才 400
	if strings.TrimSpace(req.Prompt) == "" && req.SourceImageID != nil {
		var img models.Image
		if err := h.db.First(&img, *req.SourceImageID).Error; err == nil && img.Prompt != "" {
			req.Prompt = img.Prompt
		}
	}
	if strings.TrimSpace(req.Prompt) == "" {
		utils.Fail(c, 400, "请提供 prompt（或先选中一张已解析的产品原图）")
		return
	}
	if req.Width == 0 {
		req.Width = 1024
	}
	if req.Height == 0 {
		req.Height = 1024
	}
	// 风格合并：优先用英文提示词（给模型的），空时回退到老 Prompt 字段，兼容历史数据
	var styleName string
	if req.StyleID != nil {
		var sp models.StylePreset
		if err := h.db.First(&sp, *req.StyleID).Error; err == nil {
			styleName = sp.Name
			stylePrompt := sp.PromptEN
			if stylePrompt == "" {
				stylePrompt = sp.Prompt
			}
			if strings.TrimSpace(stylePrompt) != "" {
				req.Prompt = req.Prompt + ", " + stylePrompt
			}
		}
	}
	taskID := uuid.NewString()
	task := models.AITask{
		ID: taskID, UserID: uid.(uint), Type: "generate", Status: "running",
	}
	h.db.Create(&task)
	res, err := h.ai.Generate(c.Request.Context(), uid.(uint), services.GenerateRequest{
		SourceImageID:   req.SourceImageID,
		UseAsSubject:    req.UseAsSubject,
		ProductID:       req.ProductID,
		ModelConfigID:   req.ModelConfigID,
		StyleID:         req.StyleID,
		StyleName:       styleName,
		Prompt:          req.Prompt,
		Width:           req.Width,
		Height:          req.Height,
		PromptOptimizer: req.PromptOptimizer,
	})
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		h.db.Save(&task)
		utils.Fail(c, 500, "生成失败: "+err.Error())
		return
	}
	g := res.Gallery
	task.Status = "success"
	task.Result = fmt.Sprintf(`{"galleryId":%d,"url":"/uploads/%s"}`, g.ID, g.Filename)
	task.Progress = 100
	h.db.Save(&task)
	// AI 调用本身记一行（image_generation 通常不返回 usage，TokenUsage 为零值）
	var targetName string
	if g.ProductID != nil {
		var p models.Product
		if h.db.First(&p, *g.ProductID).Error == nil {
			targetName = productNameOf(p)
		}
	}
	var aiCfg *models.ModelConfig
	if g.ModelConfigID != nil {
		var mc models.ModelConfig
		if h.db.First(&mc, *g.ModelConfigID).Error == nil {
			aiCfg = &mc
		}
	}
	h.recordAILog(c, uid.(uint), targetName, "ai.generate", aiCfg, res.TokenUsage)
	utils.OK(c, gin.H{
		"taskId":    taskID,
		"galleryId": g.ID,
		"imageUrl":  services.BuildImageURL(h.cfg.UploadDir, g.Path),
		"modelName": g.ModelName,
		"styleName": g.StyleName,
		"status":    g.Status,
	})
}

// TaskStatus 公司共享：所有登录用户都能查任意 taskId 的状态/结果
func (h *AIHandler) TaskStatus(c *gin.Context) {
	id := c.Param("id")
	var t models.AITask
	if err := h.db.Where("id = ?", id).First(&t).Error; err != nil {
		utils.Fail(c, 404, "任务不存在")
		return
	}
	utils.OK(c, t)
}

func parseUint(s string) uint {
	var n uint
	fmt.Sscanf(s, "%d", &n)
	return n
}
