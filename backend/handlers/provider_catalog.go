package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/image-ai/backend/config"
	"github.com/image-ai/backend/models"
	"github.com/image-ai/backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ProviderInfo 预置的 provider / model 目录，用于前端下拉。
// VisionModels / ImageModels 一律为空：模型名由用户主动「获取模型列表」拉取，
// 或直接手输。后端不做任何猜默认 / 探活预填。
type ProviderInfo struct {
	Key          string   `json:"key"`
	Label        string   `json:"label"`
	BaseURL      string   `json:"baseUrl"`
	Protocol     string   `json:"protocol"`
	BuiltIn      bool     `json:"builtIn"`
	Description  string   `json:"description"`
	VisionModels []string `json:"visionModels"`
	ImageModels  []string `json:"imageModels"`
}

type ProviderCatalogHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewProviderCatalogHandler(db *gorm.DB, cfg *config.Config) *ProviderCatalogHandler {
	return &ProviderCatalogHandler{db: db, cfg: cfg}
}

func (h *ProviderCatalogHandler) List(c *gin.Context) {
	utils.OK(c, []ProviderInfo{
		{
			Key:         "minimax",
			Label:       "MiniMax（海螺）",
			BaseURL:     "https://api.minimaxi.com",
			Protocol:    "minimax",
			BuiltIn:     true,
			Description: "官方地址 https://api.minimaxi.com。生图走 /v1/image_generation（支持 subject_reference 角色参考图，aspect_ratio 按尺寸自动推断）；视觉走 OpenAI 兼容 /v1/chat/completions，可处理图片/视频。",
			VisionModels: []string{},
			ImageModels:  []string{},
		},
		{
			Key:         "custom",
			Label:       "自定义（OpenAI 兼容）",
			BaseURL:     "",
			Protocol:    "openai",
			BuiltIn:     false,
			Description: "私有部署或第三方兼容网关（OpenAI / DeepSeek / 智谱 / DashScope 等），需要手动填写 BaseURL 和模型名。",
			VisionModels: []string{},
			ImageModels:  []string{},
		},
	})
}

// fetchMinimaxModels 调 MiniMax 的 /v1/models（OpenAI 兼容协议），返回所有可用模型 id。
// 网络/鉴权失败时返回 nil，由调用方走静态兜底。
// 保留作为通用 OpenAI 兼容拉取的别名（已无静态兜底，失败即返回 nil）。
func fetchMinimaxModels(baseURL, apiKey string, timeout time.Duration) []string {
	return fetchOpenAICompatModels(baseURL, apiKey, timeout)
}

// fetchOpenAICompatModels 通用 OpenAI 兼容 /v1/models 拉取，baseURL/apiKey 为空返回 nil。
// 失败时把 HTTP 状态码和响应头带回来供调用方做错误提示。
func fetchOpenAICompatModels(baseURL, apiKey string, timeout time.Duration) []string {
	if baseURL == "" || apiKey == "" {
		return nil
	}
	url := strings.TrimRight(baseURL, "/") + "/v1/models"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	raw, _ := io.ReadAll(resp.Body)
	var r struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		// 兼容 {"models":[...]} 格式
		var alt struct {
			Models []string `json:"models"`
		}
		if err := json.Unmarshal(raw, &alt); err == nil && len(alt.Models) > 0 {
			return alt.Models
		}
		return nil
	}
	out := make([]string, 0, len(r.Data))
	for _, m := range r.Data {
		if m.ID != "" {
			out = append(out, m.ID)
		}
	}
	return out
}

// fetchSingle 拉取单个 provider 的模型并按 type 分组；type 留空时返回 all + 分组。
// err 字段为非空时表示拉取失败，前端应展示给用户。
func (h *ProviderCatalogHandler) fetchSingle(baseURL, apiKey, t string) (image, vision, all []string, errMsg string) {
	models := fetchOpenAICompatModels(baseURL, apiKey, 5*time.Second)
	if len(models) == 0 {
		return nil, nil, nil, "无法从远端获取模型列表（鉴权失败 / 网络超时 / 端点未提供 /v1.models）"
	}
	img, vis := splitMinimaxModels(models)
	switch t {
	case "image":
		return img, nil, models, ""
	case "vision":
		return nil, vis, models, ""
	default:
		return img, vis, models, ""
	}
}

// FetchModels POST /api/providers/fetch-models
// 入参 { provider, baseUrl, apiKey, type }
// 出参 { all: [...], image: [...], vision: [...] }；type 缺省时 image/vision 为空，前端看 all
func (h *ProviderCatalogHandler) FetchModels(c *gin.Context) {
	var req struct {
		Provider string `json:"provider"`
		BaseURL  string `json:"baseUrl"`
		APIKey   string `json:"apiKey"`
		Type     string `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, 400, "参数错误")
		return
	}
	// 内置 provider 用官方 baseUrl 兜底
	if req.Provider == "minimax" && req.BaseURL == "" {
		req.BaseURL = "https://api.minimaxi.com"
	}
	// 编辑模式下 apiKey 留空 → 复用 DB 里现有 key
	if req.APIKey == "" {
		if req.Provider == "minimax" {
			var existing models.ModelConfig
			if err := h.db.Where("provider = ? AND enabled = ? AND api_key != ''", "minimax", true).
				Order("id desc").First(&existing).Error; err == nil {
				req.APIKey = existing.APIKey
			}
		}
		if req.APIKey == "" {
			utils.Fail(c, 400, "请先填写 API Key")
			return
		}
	}
	img, vis, all, errMsg := h.fetchSingle(req.BaseURL, req.APIKey, req.Type)
	if errMsg != "" {
		utils.Fail(c, 502, errMsg)
		return
	}
	utils.OK(c, gin.H{
		"all":    all,
		"image":  img,
		"vision": vis,
	})
}

// splitMinimaxModels 把拉到的全模型列表按名称启发式分成「生图 / 视觉」。
// MiniMax 的 model id 包含 image- 前缀的是生图，abab/MiniMax-Text/MiniMax-M 是文本/视觉。
// 实在分不出来的丢到视觉组（视觉覆盖范围更大）。
func splitMinimaxModels(models []string) (image, vision []string) {
	for _, m := range models {
		lower := strings.ToLower(m)
		switch {
		case strings.HasPrefix(lower, "image-"):
			image = append(image, m)
		case strings.HasPrefix(lower, "abab"), strings.HasPrefix(m, "MiniMax-Text"),
			strings.HasPrefix(m, "MiniMax-M"):
			vision = append(vision, m)
		default:
			// 不认识的前缀默认放视觉组（视觉能力是文本模型的子集）
			vision = append(vision, m)
		}
	}
	return
}
