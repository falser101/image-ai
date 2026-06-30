package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/image-ai/backend/config"
	"github.com/image-ai/backend/models"

	"gorm.io/gorm"
)

type AIService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewAIService(db *gorm.DB, cfg *config.Config) *AIService {
	return &AIService{db: db, cfg: cfg}
}

// 视觉模型入参限制：base64 data URL 加上 JSON 包装后整体大小有限，
// MiniMax 的 nginx 在 ~1MB 就会 413，所以本地压到 1024px / JPEG 质量 85 后通常 < 500KB
const (
	visionMaxSide  = 1024              // 最长边像素
	visionMaxBytes = 1 * 1024 * 1024   // 1MB；超过就再降质量重压
	visionJPEGQual = 85
)

// AnalyzeResult 解析图片得到的结果
type AnalyzeResult struct {
	SellingPoints []string `json:"sellingPoints"`
	Prompt        string   `json:"prompt"`
	ProductName   string   `json:"productName"`
	// TokenUsage 本次调用所消耗的 token（来自上游响应 usage；为零表示上游未返回或非计费接口）
	TokenUsage TokenUsage `json:"tokenUsage"`
}

// TokenUsage 记录一次 AI 调用的 token 消耗；任一字段可空
type TokenUsage struct {
	Prompt     int `json:"prompt"`
	Completion int `json:"completion"`
	Total      int `json:"total"`
}

// Analyze 用视觉模型分析产品图，输出卖点与生图prompt
//   - modelConfigID 非 nil 时用指定配置；否则自动选最新启用的视觉配置
//   - productName 不为空时作为强参考注入到 user 消息，帮助模型校验「图与名是否一致」
//     （同一张图，当用户记录的产品名是 A 不应识别成 B）。仅产品表里某行有 Name 时传。
func (s *AIService) Analyze(ctx context.Context, userID uint, imageID uint, modelConfigID *uint, productName string) (*AnalyzeResult, *models.ModelConfig, error) {
	var img models.Image
	if err := s.db.First(&img, imageID).Error; err != nil {
		return nil, nil, err
	}
	if img.UserID != userID {
		return nil, nil, errors.New("无权访问该图片")
	}
	// 选模型：用户指定优先；指定错（不存在/未启用/类型不对）报错
	var cfg *models.ModelConfig
	if modelConfigID != nil {
		var c models.ModelConfig
		if err := s.db.First(&c, *modelConfigID).Error; err != nil {
			return nil, nil, fmt.Errorf("指定的视觉模型(id=%d)不存在", *modelConfigID)
		}
		if !c.Enabled {
			return nil, nil, fmt.Errorf("指定的视觉模型「%s」已停用", c.Name)
		}
		if c.Type != "vision" {
			return nil, nil, fmt.Errorf("指定的模型「%s」类型不是 vision", c.Name)
		}
		cfg = &c
	} else {
		c, err := s.pickModel("vision")
		if err != nil {
			return nil, nil, errors.New("未配置可用的视觉模型，请先到「模型配置」添加")
		}
		cfg = c
	}
	if cfg.APIKey == "" {
		return nil, cfg, fmt.Errorf("视觉模型「%s」未配置 API Key", cfg.Name)
	}

	// 读图：解码后按需缩到最长边 ≤ visionMaxSide，再编码成 JPEG，避免远端 base64 data URL 触发 413
	rawData, err := os.ReadFile(img.Path)
	if err != nil {
		return nil, cfg, err
	}
	b64, mimeType, err := encodeImageForVision(rawData, visionMaxSide, visionMaxBytes)
	if err != nil {
		return nil, cfg, fmt.Errorf("预处理图片失败: %w", err)
	}

	// system 指令：要求纯 JSON、中文卖点、中文生图 prompt（按要素结构化）。
	// 单独放 system 比塞进 user 更稳，模型不会因为图片描述把指令冲掉。
	// 模板由管理员在「系统管理 → 提示词配置」维护，无内存缓存，改完立即生效。
	systemInstruction, err := GetSystemInstruction(s.db)
	if err != nil {
		return nil, cfg, fmt.Errorf("读取提示词失败: %w", err)
	}

	// user prompt：图 + 文本。文本里把"产品名 hint"塞进去，让模型把图与已知的名
	// 字比对 — 名称一致就保留，明显冲突就按实际图识别并在 prompt 里写明真实品类。
	hintLine := ""
	if pname := strings.TrimSpace(productName); pname != "" {
		hintLine = fmt.Sprintf("用户告诉我们的产品名是：「%s」。请把它作为强参考：若图里商品明显就是这个品类（外观/材质/用途都吻合），直接沿用此名；若图与该名明显冲突（比如名字是「蓝牙耳机」但图是陶瓷茶壶），请按图真实内容判定并在返回的 productName 字段里给出你看到的实际品类名称。\n\n", pname)
	}
	userText := hintLine + "请按 system 指令严格输出 JSON。"
	payload := map[string]any{
		"model": cfg.ModelName,
		"messages": []map[string]any{
			{"role": "system", "content": systemInstruction},
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "image_url", "image_url": map[string]string{"url": fmt.Sprintf("data:%s;base64,%s", mimeType, b64)}},
					{"type": "text", "text": userText},
				},
			},
		},
	}
	body, _ := json.Marshal(payload)
	// OpenAI 兼容；MiniMax 的兼容端点同样挂在 /v1/ 下（与 /v1/image_generation、/v1/models 一致）
	url := strings.TrimRight(cfg.BaseURL, "/") + "/v1/chat/completions"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, cfg, fmt.Errorf("调用视觉模型网络错误: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, cfg, fmt.Errorf("视觉模型返回 HTTP %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	var ar struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &ar); err != nil {
		return nil, cfg, fmt.Errorf("视觉模型响应 JSON 解析失败: %w", err)
	}
	if len(ar.Choices) == 0 {
		return nil, cfg, errors.New("视觉模型响应里没有 choices")
	}
	content := ar.Choices[0].Message.Content
	res, err := parseAnalyzeJSON(content)
	if err != nil {
		return nil, cfg, fmt.Errorf("视觉模型返回内容无法解析为 JSON: %w (原始: %s)", err, truncate(content, 200))
	}
	res.TokenUsage = TokenUsage{
		Prompt:     ar.Usage.PromptTokens,
		Completion: ar.Usage.CompletionTokens,
		Total:      ar.Usage.TotalTokens,
	}
	return res, cfg, nil
}

// parseAnalyzeJSON 把模型返回的文本解析成 AnalyzeResult。
// 容忍以下包装/前缀：
//   1) `<think>...</think>` 思考链（MiniMax 等模型常见；未闭合也直接剥到末尾）
//   2) ```json ... ``` 或 ``` ... ``` 代码块
//   3) 上面剥完后若还不是合法 JSON，扫描第一个 `{` 用流式解码找第一个完整对象
func parseAnalyzeJSON(s string) (*AnalyzeResult, error) {
	s = strings.TrimSpace(s)
	// 1) 剥 <think>...</think>（含未闭合情况）
	for {
		open := strings.Index(s, "<think>")
		if open < 0 {
			break
		}
		close := strings.Index(s[open:], "</think>")
		if close >= 0 {
			s = s[:open] + s[open+close+len("</think>"):]
		} else {
			// 没闭合，剥到末尾
			s = s[:open]
		}
		s = strings.TrimSpace(s)
	}
	// 2) 剥 ```json ... ``` / ``` ... ``` 包裹
	if strings.HasPrefix(s, "```") {
		// 跳过第一行（可能是 ```json 或 ```）
		if nl := strings.Index(s, "\n"); nl >= 0 {
			s = s[nl+1:]
		} else {
			s = strings.TrimPrefix(s, "```")
		}
		// 去掉尾部 ```
		s = strings.TrimSpace(s)
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	// 3) 先整体试一次
	var r AnalyzeResult
	if err := json.Unmarshal([]byte(s), &r); err == nil {
		return &r, nil
	}
	// 4) 扫描第一个 `{` 用流式解码取第一个完整对象（忽略前后乱码）
	if idx := strings.Index(s, "{"); idx >= 0 {
		dec := json.NewDecoder(strings.NewReader(s[idx:]))
		var raw json.RawMessage
		if err := dec.Decode(&raw); err == nil {
			if err := json.Unmarshal(raw, &r); err == nil {
				return &r, nil
			}
		}
	}
	return nil, errors.New("未在响应中找到可解析的 JSON 对象")
}

// GenerateResult 直接返回 Gallery
type GenerateResult struct {
	Gallery    *models.Gallery
	TokenUsage TokenUsage // image_generation 通常不返回 usage，零值合理
}

// Generate 用生图模型生成图片；任何错误（无模型/无 Key/远程失败）都直接返回 error
func (s *AIService) Generate(ctx context.Context, userID uint, req GenerateRequest) (*GenerateResult, error) {
	cfg, err := s.pickModel("image")
	if err != nil {
		return nil, errors.New("未配置可用的生图模型，请先到「模型配置」添加")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("生图模型「%s」未配置 API Key", cfg.Name)
	}
	g, err := s.callImageAPI(ctx, cfg, req, userID)
	if err != nil {
		return nil, err
	}
	return &GenerateResult{Gallery: g}, nil
}

// callImageAPI 调用真实生图接口（按 provider 分发）
func (s *AIService) callImageAPI(ctx context.Context, cfg *models.ModelConfig, req GenerateRequest, userID uint) (*models.Gallery, error) {
	switch strings.ToLower(cfg.Provider) {
	case "minimax", "hailuo":
		return s.callMinimaxImage(ctx, cfg, req, userID)
	default:
		return s.callOpenAIImage(ctx, cfg, req, userID)
	}
}

// callOpenAIImage 通用 OpenAI 兼容协议
func (s *AIService) callOpenAIImage(ctx context.Context, cfg *models.ModelConfig, req GenerateRequest, userID uint) (*models.Gallery, error) {
	payload := map[string]any{
		"model":  cfg.ModelName,
		"prompt": req.Prompt,
		"n":      1,
		"size":   fmt.Sprintf("%dx%d", req.Width, req.Height),
	}
	body, _ := json.Marshal(payload)
	url := strings.TrimRight(cfg.BaseURL, "/") + "/images/generations"
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("调用生图接口网络错误: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("生图接口返回 HTTP %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	var ir struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &ir); err != nil {
		return nil, fmt.Errorf("生图响应 JSON 解析失败: %w", err)
	}
	if len(ir.Data) == 0 {
		return nil, errors.New("生图响应里没有 data")
	}
	// 保存结果图
	filename := fmt.Sprintf("gen_%d_%s.png", time.Now().UnixNano(), randomHex(6))
	relPath := GeneratedImageRelPath(req.ProductID, filename)
	outPath := ResolveUploadPath(s.cfg.UploadDir, relPath)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return nil, fmt.Errorf("创建生成图目录失败: %w", err)
	}
	if err := downloadTo(ir.Data[0].URL, ir.Data[0].B64JSON, outPath); err != nil {
		return nil, fmt.Errorf("下载生成图失败: %w", err)
	}
	return s.saveGeneratedFile(cfg, req, userID, outPath), nil
}

// callMinimaxImage 适配 MiniMax / 海螺 image_generation 接口
// POST {base}/v1/image_generation
// body: { model, prompt, aspect_ratio, response_format, n, prompt_optimizer?, subject_reference?, style? }
// resp: { id, data: { image_urls | image_base64 }, metadata, base_resp: { status_code, status_msg } }
// 注意：业务错误（如 2013 参数错误）走 HTTP 200 + base_resp.status_code != 0，需要单独判断
func (s *AIService) callMinimaxImage(ctx context.Context, cfg *models.ModelConfig, req GenerateRequest, userID uint) (*models.Gallery, error) {
	url := strings.TrimRight(cfg.BaseURL, "/") + "/v1/image_generation"
	payload := map[string]any{
		"model":           cfg.ModelName,
		"prompt":          req.Prompt,
		"aspect_ratio":    pickAspectRatio(req.Width, req.Height),
		"response_format": "url",
		"n":               1,
	}
	// prompt_optimizer 文档默认 false，opt-in 时再打开
	if req.PromptOptimizer {
		payload["prompt_optimizer"] = true
	}
	// 如果前端请求带 sourceImageId + useAsSubject=true，则用作品作为 subject_reference
	if req.SourceImageID != nil && req.UseAsSubject {
		ref, err := s.buildSubjectReference(*req.SourceImageID)
		if err != nil {
			return nil, fmt.Errorf("构建角色参考图失败: %w", err)
		}
		if ref != nil {
			payload["subject_reference"] = []map[string]string{*ref}
		}
	}
	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("调用生图接口网络错误: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("生图接口返回 HTTP %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	// 业务错误优先：base_resp.status_code != 0 表示请求失败，status_msg 是真实原因
	var br struct {
		BaseResp struct {
			StatusCode int    `json:"status_code"`
			StatusMsg  string `json:"status_msg"`
		} `json:"base_resp"`
		Data struct {
			ImageURLs   []string `json:"image_urls"`
			ImageBase64 []string `json:"image_base64"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &br); err != nil {
		return nil, fmt.Errorf("生图响应 JSON 解析失败: %w", err)
	}
	if br.BaseResp.StatusCode != 0 {
		// 优先用 status_msg，没有时回退到 raw
		msg := br.BaseResp.StatusMsg
		if msg == "" {
			msg = truncate(string(raw), 200)
		}
		return nil, fmt.Errorf("MiniMax 业务错误(code=%d): %s", br.BaseResp.StatusCode, msg)
	}
	var imageURL, imageB64 string
	if len(br.Data.ImageURLs) > 0 {
		imageURL = br.Data.ImageURLs[0]
	}
	if imageURL == "" && len(br.Data.ImageBase64) > 0 {
		imageB64 = br.Data.ImageBase64[0]
	}
	if imageURL == "" && imageB64 == "" {
		return nil, errors.New("生图响应里没有 image_urls/image_base64: " + truncate(string(raw), 200))
	}
	filename := fmt.Sprintf("gen_%d_%s.png", time.Now().UnixNano(), randomHex(6))
	relPath := GeneratedImageRelPath(req.ProductID, filename)
	outPath := ResolveUploadPath(s.cfg.UploadDir, relPath)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return nil, fmt.Errorf("创建生成图目录失败: %w", err)
	}
	if err := downloadTo(imageURL, imageB64, outPath); err != nil {
		return nil, fmt.Errorf("下载生成图失败: %w", err)
	}
	return s.saveGeneratedFile(cfg, req, userID, outPath), nil
}

// saveGeneratedFile 把下载好的生图落盘为 Gallery 记录
func (s *AIService) saveGeneratedFile(cfg *models.ModelConfig, req GenerateRequest, userID uint, outPath string) *models.Gallery {
	st, _ := os.Stat(outPath)
	gallery := &models.Gallery{
		UserID:        userID,
		ProductID:     req.ProductID,
		SourceImageID: req.SourceImageID,
		Filename:      filepath.Base(outPath),
		Path:          outPath,
		URL:           BuildImageURL(s.cfg.UploadDir, outPath),
		Prompt:        req.Prompt,
		ModelConfigID: &cfg.ID,
		ModelName:     cfg.Name,
		StyleID:       req.StyleID,
		StyleName:     req.StyleName,
		Size:          st.Size(),
		Width:         req.Width,
		Height:        req.Height,
		Status:        "success",
	}
	if err := s.db.Create(gallery).Error; err != nil {
		return gallery
	}
	return gallery
}

// buildSubjectReference 把本地图转成 MiniMax subject_reference 用的 image_file
// 用 data:base64 内嵌，避免外网拉不到内网图片
// MiniMax 官方约束：type 当前只接受 "character"（人像），其它值会返回 status_code=2013
// 公司共享模式：允许用任意员工的原图作为参考。
func (s *AIService) buildSubjectReference(imageID uint) (*map[string]string, error) {
	var img models.Image
	if err := s.db.First(&img, imageID).Error; err != nil {
		return nil, err
	}
	data, err := os.ReadFile(img.Path)
	if err != nil {
		return nil, err
	}
	if len(data) > 10*1024*1024 {
		return nil, errors.New("参考图大于 10MB，MiniMax 接口拒绝")
	}
	mimeType := img.MimeType
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(img.Path))
	}
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	return &map[string]string{
		"type":       "character",
		"image_file": "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data),
	}, nil
}

// pickAspectRatio 把 WxH 映射到 MiniMax 支持的宽高比
// 文档合法值：1:1, 16:9, 4:3, 3:2, 2:3, 3:4, 9:16, 21:9
func pickAspectRatio(w, h int) string {
	if w <= 0 || h <= 0 {
		return "1:1"
	}
	ratio := float64(w) / float64(h)
	candidates := []struct {
		r  float64
		ar string
	}{
		{0.5625, "9:16"},
		{0.6667, "2:3"},
		{0.75, "3:4"},
		{1.0, "1:1"},
		{1.3333, "4:3"},
		{1.5, "3:2"},
		{1.7778, "16:9"},
		{2.3333, "21:9"},
	}
	best := candidates[0]
	bestDiff := abs(ratio - best.r)
	for _, c := range candidates[1:] {
		if d := abs(ratio - c.r); d < bestDiff {
			best = c
			bestDiff = d
		}
	}
	return best.ar
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

type GenerateRequest struct {
	SourceImageID   *uint
	UseAsSubject    bool   // true 时把 SourceImageID 作为 MiniMax subject_reference (type=character) 传入
	ProductID       *uint
	ModelConfigID   *uint
	StyleID         *uint
	StyleName       string
	Prompt          string
	Width           int
	Height          int
	PromptOptimizer bool
}

func (s *AIService) pickModel(t string) (*models.ModelConfig, error) {
	var cfg models.ModelConfig
	q := s.db.Where("enabled = ?", true)
	if t != "" {
		q = q.Where("type = ?", t)
	}
	if err := q.Order("id desc").First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func downloadTo(url, b64, out string) error {
	if b64 != "" {
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return err
		}
		return os.WriteFile(out, data, 0o644)
	}
	if url == "" {
		return errors.New("empty url")
	}
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// encodeImageForVision 把上传的原始图缩到 visionMaxSide 之内、再编码成 JPEG 并 base64。
// 如果编码后还 > visionMaxBytes 就再缩尺寸 + 降质量重试，最多 4 次。
// 返回 base64 字符串、最终 mime、错误。
func encodeImageForVision(raw []byte, maxSide, maxBytes int) (string, string, error) {
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", "", fmt.Errorf("解码图片失败: %w", err)
	}
	resized := imaging.Fit(src, maxSide, maxSide, imaging.Lanczos)
	quality := visionJPEGQual
	for range 4 {
		var buf bytes.Buffer
		if err := imaging.Encode(&buf, resized, imaging.JPEG, imaging.JPEGQuality(quality)); err != nil {
			return "", "", fmt.Errorf("编码 JPEG 失败: %w", err)
		}
		if buf.Len() <= maxBytes {
			return base64.StdEncoding.EncodeToString(buf.Bytes()), "image/jpeg", nil
		}
		// 体积超了：再缩 + 降质量
		maxSide = max(256, int(float64(maxSide)*0.75))
		resized = imaging.Fit(src, maxSide, maxSide, imaging.Lanczos)
		quality = max(40, quality-15)
	}
	// 走到这里说明 4 次都压不下去；用最后一次的结果返回（base64 仍可能偏大，但总比直接 413 好）
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, resized, imaging.JPEG, imaging.JPEGQuality(quality)); err != nil {
		return "", "", fmt.Errorf("编码 JPEG 失败: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), "image/jpeg", nil
}

func randomHex(n int) string {
	const letters = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Microsecond)
	}
	return string(b)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
