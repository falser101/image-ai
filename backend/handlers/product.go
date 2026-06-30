package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/image-ai/backend/config"
	"github.com/image-ai/backend/models"
	"github.com/image-ai/backend/services"
	"github.com/image-ai/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewProductHandler(db *gorm.DB, cfg *config.Config) *ProductHandler {
	return &ProductHandler{db: db, cfg: cfg}
}

type createProductReq struct {
	Name    string `json:"name"`
	ImageID uint   `json:"imageId"`
}

// Create 创建产品。公司共享模式下对全公司同名产品去重。
//   - Name trim 后非空，且长度 1..128
//   - Name 在 products 表里唯一（unscope，软删的不算）
//   - 冲突返回 1005「产品名已存在」
//   - 空名返回 400「产品名不能为空」
func (h *ProductHandler) Create(c *gin.Context) {
	uid, _ := c.Get("userId")
	var req createProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, 400, "参数错误")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		utils.Fail(c, 400, "产品名不能为空")
		return
	}
	if len(name) > 128 {
		utils.Fail(c, 400, "产品名不能超过 128 字符")
		return
	}
	var count int64
	if err := h.db.Unscoped().Model(&models.Product{}).
		Where("name = ?", name).
		Count(&count).Error; err != nil {
		utils.Fail(c, 500, "校验失败")
		return
	}
	if count > 0 {
		utils.Fail(c, 1005, "产品名已存在：「"+name+"」")
		return
	}
	p := models.Product{UserID: uid.(uint), Name: name}
	if req.ImageID > 0 {
		p.ImageID = &req.ImageID
	}
	if err := h.db.Create(&p).Error; err != nil {
		utils.Fail(c, 500, "创建失败")
		return
	}
	utils.OK(c, p)
}

// CheckName 看 name 是否被占用。前端实时去重（debounce 调）；不区分大小写无关，全等。
// GET /api/products/check-name?name=xxx
func (h *ProductHandler) CheckName(c *gin.Context) {
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		utils.OK(c, gin.H{"name": name, "exists": false, "valid": false})
		return
	}
	if len(name) > 128 {
		utils.OK(c, gin.H{"name": name, "exists": false, "valid": false, "reason": "产品名不能超过 128 字符"})
		return
	}
	var count int64
	h.db.Unscoped().Model(&models.Product{}).Where("name = ?", name).Count(&count)
	utils.OK(c, gin.H{"name": name, "exists": count > 0, "valid": count == 0})
}

// ProductListItem 在 Product 基础上附带生成图统计：
//   GalleryCount       该产品下生成图总数
//   PreviewGalleryURL  最新一张生成图 URL（用于 Products 页跳转预览）
//   CoverImageURL      该产品 cover 原图的 URL（用于 Products 列表缩略）
//   SourceImageCount   该产品下的原图张数
type ProductListItem struct {
	models.Product
	GalleryCount      int    `json:"galleryCount"`
	PreviewGalleryURL string `json:"previewGalleryUrl,omitempty"`
	CoverImageURL     string `json:"coverImageUrl,omitempty"`
	SourceImageCount  int    `json:"sourceImageCount"`
}

// fillGalleryStats 批量补齐 list 中每个产品的 GalleryCount / PreviewGalleryURL / CoverImageURL / SourceImageCount。
// 公司共享模式：跨员工统计，不按 user_id 过滤。
// 用两条查询（一次 count 聚合 + 一次按 id desc 取最新），避免 N+1。
func (h *ProductHandler) fillGalleryStats(products []models.Product) (countMap map[uint]int, previewMap map[uint]string, coverMap map[uint]string, sourceCountMap map[uint]int) {
	countMap = map[uint]int{}
	previewMap = map[uint]string{}
	coverMap = map[uint]string{}
	sourceCountMap = map[uint]int{}
	ids := make([]uint, 0, len(products))
	for _, p := range products {
		ids = append(ids, p.ID)
	}
	if len(ids) == 0 {
		return
	}
	// 1) 每个 product 的 gallery 数
	type row struct {
		ProductID uint
		Cnt       int
	}
	var rows []row
	h.db.Model(&models.Gallery{}).
		Select("product_id as product_id, count(*) as cnt").
		Where("product_id IN ?", ids).
		Group("product_id").
		Scan(&rows)
	for _, r := range rows {
		countMap[r.ProductID] = r.Cnt
	}
	// 2) 每个 product 的最新一张 gallery（id desc 取首条）
	var gs []models.Gallery
	h.db.Model(&models.Gallery{}).
		Where("product_id IN ?", ids).
		Order("id desc").
		Find(&gs)
	for _, g := range gs {
		if g.ProductID == nil {
			continue
		}
		if _, exists := previewMap[*g.ProductID]; !exists {
			previewMap[*g.ProductID] = g.URL
		}
	}
	// 3) 原图统计 + cover 图 URL
	var imgs []models.Image
	h.db.Model(&models.Image{}).
		Where("product_id IN ?", ids).
		Order("id asc").
		Find(&imgs)
	for _, img := range imgs {
		if img.ProductID == nil {
			continue
		}
		sourceCountMap[*img.ProductID]++
		// cover 兜底：id asc 的第一张
		if _, exists := coverMap[*img.ProductID]; !exists {
			coverMap[*img.ProductID] = services.BuildImageURL(h.cfg.UploadDir, img.Path)
		}
	}
	// cover 优先用 Product.ImageID 指向的那张
	for _, p := range products {
		if p.ImageID != nil {
			for _, img := range imgs {
				if img.ID == *p.ImageID && img.ProductID != nil && *img.ProductID == p.ID {
					coverMap[p.ID] = services.BuildImageURL(h.cfg.UploadDir, img.Path)
					break
				}
			}
		}
	}
	return
}

// List 公司共享：所有人能看到所有产品
func (h *ProductHandler) List(c *gin.Context) {
	var products []models.Product
	h.db.Order("id desc").Find(&products)

	countMap, previewMap, coverMap, sourceCountMap := h.fillGalleryStats(products)
	out := make([]ProductListItem, len(products))
	for i, p := range products {
		out[i] = ProductListItem{
			Product:           p,
			GalleryCount:      countMap[p.ID],
			PreviewGalleryURL: previewMap[p.ID],
			CoverImageURL:     coverMap[p.ID],
			SourceImageCount:  sourceCountMap[p.ID],
		}
	}
	utils.OK(c, out)
}

// Get 公司共享：所有登录用户都能看任一产品详情。
func (h *ProductHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p models.Product
	if err := h.db.First(&p, id).Error; err != nil {
		utils.Fail(c, 404, "产品不存在")
		return
	}
	// 拉取该产品下所有原图，按 id 升序（最早的上传在前）。跨员工共享。
	var imgs []models.Image
	h.db.Model(&models.Image{}).
		Where("product_id = ?", id).
		Order("id asc").
		Find(&imgs)
	sources := make([]SourceImageItem, 0, len(imgs))
	for _, img := range imgs {
		sources = append(sources, SourceImageItem{
			ID:            img.ID,
			URL:           services.BuildImageURL(h.cfg.UploadDir, img.Path),
			Prompt:        img.Prompt,
			SellingPoints: parseSellingPtsJSON(img.SellingPts),
			Analyzed:      img.Analyzed,
			CreatedAt:     img.CreatedAt,
		})
	}
	utils.OK(c, gin.H{
		"id":             p.ID,
		"userId":         p.UserID,
		"name":           p.Name,
		"imageId":        p.ImageID,
		"prompt":         p.Prompt,
		"sellingPoints":  parseSellingPtsJSON(p.SellingPts),
		"createdAt":      p.CreatedAt,
		"updatedAt":      p.UpdatedAt,
		"sourceImages":   sources,
		"galleryCount":   h.galleryCountFor(p.ID),
		"previewGallery": h.previewGalleryFor(p.ID),
	})
}

// SourceImageItem 产品详情里的原图精简视图（只挑前端要的字段，避免把 Path 之类内部信息露出去）
type SourceImageItem struct {
	ID            uint      `json:"id"`
	URL           string    `json:"url"`
	Prompt        string    `json:"prompt"`
	SellingPoints []string  `json:"sellingPoints"`
	Analyzed      bool      `json:"analyzed"`
	CreatedAt     time.Time `json:"createdAt"`
}

func parseSellingPtsJSON(s string) []string {
	if s == "" {
		return []string{}
	}
	var arr []string
	if err := json.Unmarshal([]byte(s), &arr); err != nil {
		return []string{}
	}
	return arr
}

// galleryCountFor / previewGalleryFor 公司共享：跨员工统计
func (h *ProductHandler) galleryCountFor(productID uint) int {
	var n int64
	h.db.Model(&models.Gallery{}).
		Where("product_id = ?", productID).
		Count(&n)
	return int(n)
}

func (h *ProductHandler) previewGalleryFor(productID uint) string {
	var g models.Gallery
	if err := h.db.Model(&models.Gallery{}).
		Where("product_id = ?", productID).
		Order("id desc").
		Limit(1).
		First(&g).Error; err != nil {
		return ""
	}
	return services.BuildImageURL(h.cfg.UploadDir, g.Path)
}

// Delete 员工只能删自己创建的产品；管理员可删任意
func (h *ProductHandler) Delete(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	id, _ := strconv.Atoi(c.Param("id"))
	q := h.db.Model(&models.Product{}).Where("id = ?", id)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	res := q.Delete(&models.Product{})
	if res.Error != nil {
		utils.Fail(c, 500, "删除失败")
		return
	}
	if res.RowsAffected == 0 {
		utils.Fail(c, 404, "产品不存在或无权访问")
		return
	}
	utils.OK(c, nil)
}

type updateProductReq struct {
	Name string `json:"name"`
}

// Update 修改产品名。员工只能改自己的产品；管理员可改任意。
// ImageID / Prompt / SellingPts 都是从最近一次上传反规范化出来的，不允许外部直接改。
func (h *ProductHandler) Update(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	id, _ := strconv.Atoi(c.Param("id"))
	var req updateProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, 400, "参数错误")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		utils.Fail(c, 400, "产品名不能为空")
		return
	}
	if len(name) > 128 {
		utils.Fail(c, 400, "产品名不能超过 128 字符")
		return
	}
	var p models.Product
	q := h.db.Where("id = ?", id)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	if err := q.First(&p).Error; err != nil {
		utils.Fail(c, 404, "产品不存在或无权访问")
		return
	}
	// 如果改名了，查重（排除自己）
	if p.Name != name {
		var dup int64
		if err := h.db.Unscoped().Model(&models.Product{}).
			Where("name = ? AND id <> ?", name, id).
			Count(&dup).Error; err != nil {
			utils.Fail(c, 500, "校验失败")
			return
		}
		if dup > 0 {
			utils.Fail(c, 1005, "产品名已存在：「"+name+"」")
			return
		}
	}
	p.Name = name
	if err := h.db.Save(&p).Error; err != nil {
		utils.Fail(c, 500, "更新失败")
		return
	}
	utils.OK(c, p)
}

// UploadSourceImage 给产品上传一张原图，自动跑视觉模型解析卖点 + prompt，
// 解析失败时回滚（删除落盘文件 + 删除 Image 行）。
// multipart: file（必填）, modelConfigId（可选）
func (h *ProductHandler) UploadSourceImage(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	pid := parseUint(c.Param("id"))
	if pid == 0 {
		utils.Fail(c, 400, "产品 ID 无效")
		return
	}
	// 校验产品归属
	var product models.Product
	pq := h.db.Model(&models.Product{}).Where("id = ?", pid)
	if role != "admin" {
		pq = pq.Where("user_id = ?", uid)
	}
	if err := pq.First(&product).Error; err != nil {
		utils.Fail(c, 404, "产品不存在或无权访问")
		return
	}

	// 可选：指定 vision 模型
	var modelCfgID *uint
	if s := c.PostForm("modelConfigId"); s != "" {
		v := parseUint(s)
		if v > 0 {
			modelCfgID = &v
		}
	}

	// 落盘 + 建 Image 行
	img, err := uploadImage(c, h.db, h.cfg, &pid)
	if err != nil {
		utils.Fail(c, 400, "上传失败: "+err.Error())
		return
	}
	img.ProductID = &pid

	// 跑视觉模型。失败：回滚（删文件 + 删 Image 行），让用户重试。
	// 把 product.Name 作为 hint 传进去，让 AI 把图和已知名字对齐 — 防止换图时识别错品类。
	ai := services.NewAIService(h.db, h.cfg)
	res, _, err := ai.Analyze(c.Request.Context(), uid.(uint), img.ID, modelCfgID, product.Name)
	if err != nil {
		_ = os.Remove(img.Path)
		h.db.Delete(img)
		utils.Fail(c, 500, "解析失败: "+err.Error())
		return
	}

	// 写回 Image 元数据
	jsonSP, _ := json.Marshal(res.SellingPoints)
	img.Prompt = res.Prompt
	img.SellingPts = string(jsonSP)
	img.Analyzed = true
	h.db.Save(img)

	// 同步 product 反规范化字段（最新上传的图作为 cover）
	product.Prompt = res.Prompt
	product.SellingPts = string(jsonSP)
	product.ImageID = &img.ID
	h.db.Save(&product)

	// 追加本次 AI 卖点
	for _, p := range res.SellingPoints {
		h.db.Create(&models.SellingPoint{
			UserID:    uid.(uint),
			ProductID: &pid,
			Content:   p,
			Source:    "ai",
		})
	}

	// 统计当前产品下的原图数
	var cnt int64
	h.db.Model(&models.Image{}).Where("product_id = ?", pid).Count(&cnt)

	utils.OK(c, gin.H{
		"image":         imgResp(img, h.cfg.UploadDir),
		"productId":     pid,
		"imageCount":    cnt,
		"sellingPoints": res.SellingPoints,
		"prompt":        res.Prompt,
	})
}

// imgResp 把 models.Image 渲染成给前端的精简视图（URL 是 /uploads/...，不漏 Path）
func imgResp(img *models.Image, uploadDir string) gin.H {
	var sp []string
	_ = json.Unmarshal([]byte(img.SellingPts), &sp)
	if sp == nil {
		sp = []string{}
	}
	return gin.H{
		"id":            img.ID,
		"userId":        img.UserID,
		"productId":     img.ProductID,
		"filename":      img.Filename,
		"url":           services.BuildImageURL(uploadDir, img.Path),
		"size":          img.Size,
		"mimeType":      img.MimeType,
		"prompt":        img.Prompt,
		"sellingPoints": sp,
		"analyzed":      img.Analyzed,
		"createdAt":     img.CreatedAt,
		"updatedAt":     img.UpdatedAt,
	}
}

// uploadImage 上传产品原图：自动创建 Image 记录，返回 imageId
// productID 决定落盘位置（products/{pid}/source/...）。可空，空时走 misc/source/。
func uploadImage(c *gin.Context, db *gorm.DB, cfg *config.Config, productID *uint) (*models.Image, error) {
	uid, _ := c.Get("userId")
	file, err := c.FormFile("file")
	if err != nil {
		return nil, err
	}
	ext := filepath.Ext(file.Filename)
	filename := uuid.NewString() + ext
	relPath := services.SourceImageRelPath(productID, filename)
	out := services.ResolveUploadPath(cfg.UploadDir, relPath)
	// 子目录可能不存在，先建
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return nil, err
	}
	if err := c.SaveUploadedFile(file, out); err != nil {
		return nil, err
	}
	img := &models.Image{
		UserID:   uid.(uint),
		Filename: file.Filename,
		Path:     out,
		Size:     file.Size,
		MimeType: file.Header.Get("Content-Type"),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.Create(img).Error; err != nil {
		return nil, err
	}
	return img, nil
}
