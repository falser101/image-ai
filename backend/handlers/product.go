package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
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

func (h *ProductHandler) Create(c *gin.Context) {
	uid, _ := c.Get("userId")
	var req createProductReq
	c.ShouldBindJSON(&req)
	p := models.Product{UserID: uid.(uint), Name: req.Name}
	if req.ImageID > 0 {
		p.ImageID = &req.ImageID
	}
	if err := h.db.Create(&p).Error; err != nil {
		utils.Fail(c, 500, "创建失败")
		return
	}
	utils.OK(c, p)
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

// fillGalleryStats 批量补齐 list 中每个产品的 GalleryCount / PreviewGalleryURL。
// 用两条查询（一次 count 聚合 + 一次按 id desc 取最新），避免 N+1。
func (h *ProductHandler) fillGalleryStats(uid uint, role string, products []models.Product) (countMap map[uint]int, previewMap map[uint]string, coverMap map[uint]string, sourceCountMap map[uint]int) {
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
	// 1) 每个 product 的 gallery 数（按 user_id 隔离）
	type row struct {
		ProductID uint
		Cnt       int
	}
	cq := h.db.Model(&models.Gallery{}).
		Select("product_id as product_id, count(*) as cnt").
		Where("product_id IN ?", ids)
	if role != "admin" {
		cq = cq.Where("user_id = ?", uid)
	}
	var rows []row
	cq.Group("product_id").Scan(&rows)
	for _, r := range rows {
		countMap[r.ProductID] = r.Cnt
	}
	// 2) 每个 product 的最新一张 gallery（id desc 取首条）
	pq := h.db.Model(&models.Gallery{}).
		Where("product_id IN ?", ids).
		Order("id desc")
	if role != "admin" {
		pq = pq.Where("user_id = ?", uid)
	}
	var gs []models.Gallery
	pq.Find(&gs)
	for _, g := range gs {
		if g.ProductID == nil {
			continue
		}
		if _, exists := previewMap[*g.ProductID]; !exists {
			previewMap[*g.ProductID] = g.URL
		}
	}
	// 3) 原图统计 + cover 图 URL
	iq := h.db.Model(&models.Image{}).Where("product_id IN ?", ids)
	if role != "admin" {
		iq = iq.Where("user_id = ?", uid)
	}
	var imgs []models.Image
	iq.Order("id asc").Find(&imgs)
	for _, img := range imgs {
		if img.ProductID == nil {
			continue
		}
		sourceCountMap[*img.ProductID]++
		// cover = 最早的一张（最新上传会覆盖 Product.ImageID，但 cover 仍用 product 表指向的）
		// 实际渲染时优先用 coverMap，没设过才用 id asc 的第一张兜底
		if _, exists := coverMap[*img.ProductID]; !exists {
			coverMap[*img.ProductID] = "/uploads/" + filepath.Base(img.Path)
		}
	}
	// cover 优先用 Product.ImageID 指向的那张
	for _, p := range products {
		if p.ImageID != nil {
			for _, img := range imgs {
				if img.ID == *p.ImageID && img.ProductID != nil && *img.ProductID == p.ID {
					coverMap[p.ID] = "/uploads/" + filepath.Base(img.Path)
					break
				}
			}
		}
	}
	return
}

func (h *ProductHandler) List(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	q := h.db.Model(&models.Product{})
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	var products []models.Product
	q.Order("id desc").Find(&products)

	countMap, previewMap, coverMap, sourceCountMap := h.fillGalleryStats(uid.(uint), role.(string), products)
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

func (h *ProductHandler) Get(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	id, _ := strconv.Atoi(c.Param("id"))
	q := h.db.Model(&models.Product{}).Where("id = ?", id)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	var p models.Product
	if err := q.First(&p).Error; err != nil {
		utils.Fail(c, 404, "产品不存在")
		return
	}
	// 拉取该产品下所有原图，按 id 升序（最早的上传在前）
	var imgs []models.Image
	iq := h.db.Model(&models.Image{}).Where("product_id = ?", id)
	if role != "admin" {
		iq = iq.Where("user_id = ?", uid)
	}
	iq.Order("id asc").Find(&imgs)
	sources := make([]SourceImageItem, 0, len(imgs))
	for _, img := range imgs {
		sources = append(sources, SourceImageItem{
			ID:            img.ID,
			URL:           "/uploads/" + filepath.Base(img.Path),
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
		"galleryCount":   h.galleryCountFor(uid.(uint), role.(string), p.ID),
		"previewGallery": h.previewGalleryFor(uid.(uint), role.(string), p.ID),
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

func (h *ProductHandler) galleryCountFor(uid uint, role string, productID uint) int {
	q := h.db.Model(&models.Gallery{}).Where("product_id = ?", productID)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	var n int64
	q.Count(&n)
	return int(n)
}

func (h *ProductHandler) previewGalleryFor(uid uint, role string, productID uint) string {
	q := h.db.Model(&models.Gallery{}).Where("product_id = ?", productID).Order("id desc").Limit(1)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	var g models.Gallery
	if err := q.First(&g).Error; err != nil {
		return ""
	}
	return "/uploads/" + filepath.Base(g.Path)
}

func (h *ProductHandler) Delete(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	id, _ := strconv.Atoi(c.Param("id"))
	q := h.db.Model(&models.Product{}).Where("id = ?", id)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	if err := q.Delete(&models.Product{}).Error; err != nil {
		utils.Fail(c, 500, "删除失败")
		return
	}
	utils.OK(c, nil)
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
	img, err := uploadImage(c, h.db, h.cfg)
	if err != nil {
		utils.Fail(c, 400, "上传失败: "+err.Error())
		return
	}
	img.ProductID = &pid

	// 跑视觉模型。失败：回滚（删文件 + 删 Image 行），让用户重试
	ai := services.NewAIService(h.db, h.cfg)
	res, _, err := ai.Analyze(c.Request.Context(), uid.(uint), img.ID, modelCfgID)
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
		"image":         imgResp(img),
		"productId":     pid,
		"imageCount":    cnt,
		"sellingPoints": res.SellingPoints,
		"prompt":        res.Prompt,
	})
}

// imgResp 把 models.Image 渲染成给前端的精简视图（URL 是 /uploads/...，不漏 Path）
func imgResp(img *models.Image) gin.H {
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
		"url":           "/uploads/" + filepath.Base(img.Path),
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
func uploadImage(c *gin.Context, db *gorm.DB, cfg *config.Config) (*models.Image, error) {
	uid, _ := c.Get("userId")
	file, err := c.FormFile("file")
	if err != nil {
		return nil, err
	}
	ext := filepath.Ext(file.Filename)
	filename := uuid.NewString() + ext
	out := filepath.Join(cfg.UploadDir, filename)
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
