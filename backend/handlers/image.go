package handlers

import (
	"path/filepath"
	"strconv"

	"github.com/image-ai/backend/config"
	"github.com/image-ai/backend/models"
	"github.com/image-ai/backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GalleryListItem 在 Gallery 基础上附带原图 URL（如果有 SourceImageID）
// 和所属产品名（如果有 ProductID）。用嵌入避免重复字段；JSON 会把 Gallery
// 的字段打平，再追加 sourceImageUrl / productName。
type GalleryListItem struct {
	models.Gallery
	SourceImageURL string `json:"sourceImageUrl,omitempty"`
	ProductName    string `json:"productName,omitempty"`
}

// resolveSourceImageURLs 批量查 SourceImageID 对应的 /uploads/<filename> 映射。
func (h *GalleryHandler) resolveSourceImageURLs(uid uint, role string, galleries []models.Gallery) map[uint]string {
	out := map[uint]string{}
	ids := make([]uint, 0, len(galleries))
	for _, g := range galleries {
		if g.SourceImageID != nil {
			ids = append(ids, *g.SourceImageID)
		}
	}
	if len(ids) == 0 {
		return out
	}
	q := h.db.Model(&models.Image{}).Where("id IN ?", ids)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	var imgs []models.Image
	q.Find(&imgs)
	for _, img := range imgs {
		out[img.ID] = "/uploads/" + filepath.Base(img.Path)
	}
	return out
}

// resolveProductNames 批量查 ProductID 对应的产品名映射。
// 用在 Gallery List/Get 里，给每条生成图补上所属产品名。
func (h *GalleryHandler) resolveProductNames(uid uint, role string, galleries []models.Gallery) map[uint]string {
	out := map[uint]string{}
	ids := make([]uint, 0, len(galleries))
	for _, g := range galleries {
		if g.ProductID != nil {
			ids = append(ids, *g.ProductID)
		}
	}
	if len(ids) == 0 {
		return out
	}
	q := h.db.Model(&models.Product{}).Where("id IN ?", ids)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	var ps []models.Product
	q.Find(&ps)
	for _, p := range ps {
		out[p.ID] = p.Name
	}
	return out
}

type ImageHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewImageHandler(db *gorm.DB, cfg *config.Config) *ImageHandler {
	return &ImageHandler{db: db, cfg: cfg}
}

func (h *ImageHandler) List(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	q := h.db.Model(&models.Image{})
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	if pid := c.Query("productId"); pid != "" {
		q = q.Where("product_id = ?", pid)
	}
	var list []models.Image
	q.Order("id desc").Limit(500).Find(&list)
	utils.OK(c, list)
}

func (h *ImageHandler) Get(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	id, _ := strconv.Atoi(c.Param("id"))
	q := h.db.Model(&models.Image{}).Where("id = ?", id)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	var img models.Image
	if err := q.First(&img).Error; err != nil {
		utils.Fail(c, 404, "图片不存在")
		return
	}
	utils.OK(c, img)
}

func (h *ImageHandler) Delete(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	id, _ := strconv.Atoi(c.Param("id"))
	q := h.db.Model(&models.Image{}).Where("id = ?", id)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	q.Delete(&models.Image{})
	utils.OK(c, nil)
}

func (h *ImageHandler) ServeFile(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	id, _ := strconv.Atoi(c.Param("id"))
	q := h.db.Model(&models.Image{}).Where("id = ?", id)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	var img models.Image
	if err := q.First(&img).Error; err != nil {
		utils.Fail(c, 404, "图片不存在")
		return
	}
	c.File(img.Path)
}

type GalleryHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewGalleryHandler(db *gorm.DB, cfg *config.Config) *GalleryHandler {
	return &GalleryHandler{db: db, cfg: cfg}
}

func (h *GalleryHandler) List(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	q := h.db.Model(&models.Gallery{})
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	if pid := c.Query("productId"); pid != "" {
		q = q.Where("product_id = ?", pid)
	}
	if mid := c.Query("modelConfigId"); mid != "" {
		q = q.Where("model_config_id = ?", mid)
	}
	if sid := c.Query("styleId"); sid != "" {
		q = q.Where("style_id = ?", sid)
	}
	if kw := c.Query("keyword"); kw != "" {
		q = q.Where("prompt LIKE ? OR style_name LIKE ? OR model_name LIKE ?", "%"+kw+"%", "%"+kw+"%", "%"+kw+"%")
	}
	var list []models.Gallery
	q.Order("id desc").Limit(500).Find(&list)

	// 批量补齐原图 URL + 产品名
	srcMap := h.resolveSourceImageURLs(uid.(uint), role.(string), list)
	prodMap := h.resolveProductNames(uid.(uint), role.(string), list)
	out := make([]GalleryListItem, len(list))
	for i, g := range list {
		out[i] = GalleryListItem{Gallery: g}
		if g.SourceImageID != nil {
			out[i].SourceImageURL = srcMap[*g.SourceImageID]
		}
		if g.ProductID != nil {
			out[i].ProductName = prodMap[*g.ProductID]
		}
	}
	utils.OK(c, out)
}

func (h *GalleryHandler) Get(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	id, _ := strconv.Atoi(c.Param("id"))
	q := h.db.Model(&models.Gallery{}).Where("id = ?", id)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	var g models.Gallery
	if err := q.First(&g).Error; err != nil {
		utils.Fail(c, 404, "图库项不存在")
		return
	}
	item := GalleryListItem{Gallery: g}
	if g.SourceImageID != nil {
		srcMap := h.resolveSourceImageURLs(uid.(uint), role.(string), []models.Gallery{g})
		item.SourceImageURL = srcMap[*g.SourceImageID]
	}
	if g.ProductID != nil {
		prodMap := h.resolveProductNames(uid.(uint), role.(string), []models.Gallery{g})
		item.ProductName = prodMap[*g.ProductID]
	}
	utils.OK(c, item)
}

func (h *GalleryHandler) Delete(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	id, _ := strconv.Atoi(c.Param("id"))
	q := h.db.Model(&models.Gallery{}).Where("id = ?", id)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	q.Delete(&models.Gallery{})
	utils.OK(c, nil)
}

func (h *GalleryHandler) ServeFile(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	id, _ := strconv.Atoi(c.Param("id"))
	q := h.db.Model(&models.Gallery{}).Where("id = ?", id)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	var g models.Gallery
	if err := q.First(&g).Error; err != nil {
		utils.Fail(c, 404, "图库项不存在")
		return
	}
	c.File(g.Path)
}
