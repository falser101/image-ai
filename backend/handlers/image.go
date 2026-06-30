package handlers

import (
	"strconv"

	"github.com/image-ai/backend/config"
	"github.com/image-ai/backend/models"
	"github.com/image-ai/backend/services"
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
// 公司共享：跨员工查找。
func (h *GalleryHandler) resolveSourceImageURLs(galleries []models.Gallery) map[uint]string {
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
	var imgs []models.Image
	h.db.Model(&models.Image{}).Where("id IN ?", ids).Find(&imgs)
	for _, img := range imgs {
		out[img.ID] = services.BuildImageURL(h.cfg.UploadDir, img.Path)
	}
	return out
}

// resolveProductNames 批量查 ProductID 对应的产品名映射。
// 公司共享：跨员工查找。
func (h *GalleryHandler) resolveProductNames(galleries []models.Gallery) map[uint]string {
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
	var ps []models.Product
	h.db.Model(&models.Product{}).Where("id IN ?", ids).Find(&ps)
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

// List / Get 公司共享：所有登录用户都能看
func (h *ImageHandler) List(c *gin.Context) {
	q := h.db.Model(&models.Image{})
	if pid := c.Query("productId"); pid != "" {
		q = q.Where("product_id = ?", pid)
	}
	var list []models.Image
	q.Order("id desc").Limit(500).Find(&list)
	utils.OK(c, list)
}

func (h *ImageHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var img models.Image
	if err := h.db.First(&img, id).Error; err != nil {
		utils.Fail(c, 404, "图片不存在")
		return
	}
	utils.OK(c, img)
}

// Delete 员工只能删自己上传的原图
func (h *ImageHandler) Delete(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	id, _ := strconv.Atoi(c.Param("id"))
	q := h.db.Model(&models.Image{}).Where("id = ?", id)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	res := q.Delete(&models.Image{})
	if res.RowsAffected == 0 {
		utils.Fail(c, 404, "不存在或无权访问")
		return
	}
	utils.OK(c, nil)
}

// ServeFile 公司共享：所有登录用户都能下载
func (h *ImageHandler) ServeFile(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var img models.Image
	if err := h.db.First(&img, id).Error; err != nil {
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

// List 公司共享：所有登录用户都能看
func (h *GalleryHandler) List(c *gin.Context) {
	q := h.db.Model(&models.Gallery{})
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
	srcMap := h.resolveSourceImageURLs(list)
	prodMap := h.resolveProductNames(list)
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

// Get 公司共享：所有登录用户都能看
func (h *GalleryHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var g models.Gallery
	if err := h.db.First(&g, id).Error; err != nil {
		utils.Fail(c, 404, "图库项不存在")
		return
	}
	item := GalleryListItem{Gallery: g}
	if g.SourceImageID != nil {
		srcMap := h.resolveSourceImageURLs([]models.Gallery{g})
		item.SourceImageURL = srcMap[*g.SourceImageID]
	}
	if g.ProductID != nil {
		prodMap := h.resolveProductNames([]models.Gallery{g})
		item.ProductName = prodMap[*g.ProductID]
	}
	utils.OK(c, item)
}

// Delete 员工只能删自己生成的图
func (h *GalleryHandler) Delete(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	id, _ := strconv.Atoi(c.Param("id"))
	q := h.db.Model(&models.Gallery{}).Where("id = ?", id)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	res := q.Delete(&models.Gallery{})
	if res.RowsAffected == 0 {
		utils.Fail(c, 404, "不存在或无权访问")
		return
	}
	utils.OK(c, nil)
}

// ServeFile 公司共享：所有登录用户都能下载
func (h *GalleryHandler) ServeFile(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var g models.Gallery
	if err := h.db.First(&g, id).Error; err != nil {
		utils.Fail(c, 404, "图库项不存在")
		return
	}
	c.File(g.Path)
}
