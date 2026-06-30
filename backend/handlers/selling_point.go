package handlers

import (
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/image-ai/backend/config"
	"github.com/image-ai/backend/models"
	"github.com/image-ai/backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SellingPointHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewSellingPointHandler(db *gorm.DB, cfg *config.Config) *SellingPointHandler {
	return &SellingPointHandler{db: db, cfg: cfg}
}

type createPointReq struct {
	Content  string `json:"content" binding:"required"`
	Source   string `json:"source"`
	ImageID  *uint  `json:"imageId"`
	SaveAsProduct bool `json:"saveAsProduct"`
}

func (h *SellingPointHandler) CreateForProduct(c *gin.Context) {
	uid, _ := c.Get("userId")
	pid64, _ := strconv.Atoi(c.Param("id"))
	pid := uint(pid64)
	var req createPointReq
	c.ShouldBindJSON(&req)
	pts := splitLines(req.Content)
	if len(pts) == 0 {
		utils.Fail(c, 400, "卖点内容不能为空")
		return
	}
	var created []models.SellingPoint
	for _, p := range pts {
		sp := models.SellingPoint{
			UserID:    uid.(uint),
			ProductID: &pid,
			Content:   p,
			Source:    orDefault(req.Source, "manual"),
		}
		h.db.Create(&sp)
		created = append(created, sp)
	}
	utils.OK(c, created)
}

// List / ListByProduct / Get 公司共享：所有登录用户都能看
func (h *SellingPointHandler) List(c *gin.Context) {
	q := h.db.Model(&models.SellingPoint{})
	if pid := c.Query("productId"); pid != "" {
		q = q.Where("product_id = ?", pid)
	}
	if src := c.Query("source"); src != "" {
		q = q.Where("source = ?", src)
	}
	var list []models.SellingPoint
	q.Order("id desc").Limit(500).Find(&list)
	utils.OK(c, list)
}

func (h *SellingPointHandler) ListByProduct(c *gin.Context) {
	pid, _ := strconv.Atoi(c.Param("id"))
	var list []models.SellingPoint
	h.db.Model(&models.SellingPoint{}).
		Where("product_id = ?", pid).
		Order("id asc").
		Find(&list)
	utils.OK(c, list)
}

func (h *SellingPointHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p models.SellingPoint
	if err := h.db.First(&p, id).Error; err != nil {
		utils.Fail(c, 404, "不存在")
		return
	}
	utils.OK(c, p)
}

// Delete 员工只能删自己创建的卖点
func (h *SellingPointHandler) Delete(c *gin.Context) {
	uid, _ := c.Get("userId")
	role, _ := c.Get("role")
	id, _ := strconv.Atoi(c.Param("id"))
	q := h.db.Model(&models.SellingPoint{}).Where("id = ?", id)
	if role != "admin" {
		q = q.Where("user_id = ?", uid)
	}
	res := q.Delete(&models.SellingPoint{})
	if res.RowsAffected == 0 {
		utils.Fail(c, 404, "不存在或无权访问")
		return
	}
	utils.OK(c, nil)
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	// 接受 JSON 数组或换行/逗号分隔
	s = trim(s)
	if len(s) > 0 && s[0] == '[' {
		var arr []string
		_ = json.Unmarshal([]byte(s), &arr)
		return arr
	}
	var out []string
	cur := ""
	for _, ch := range s {
		if ch == '\n' || ch == '；' || ch == ';' {
			if cur != "" {
				out = append(out, trim(cur))
				cur = ""
			}
			continue
		}
		cur += string(ch)
	}
	if cur != "" {
		out = append(out, trim(cur))
	}
	return out
}

func trim(s string) string {
	for len(s) > 0 {
		c := s[0]
		if c != ' ' && c != '\t' && c != '\r' && c != ',' {
			break
		}
		s = s[1:]
	}
	for len(s) > 0 {
		c := s[len(s)-1]
		if c != ' ' && c != '\t' && c != '\r' && c != ',' {
			break
		}
		s = s[:len(s)-1]
	}
	// 去掉首尾中英文句号
	for len(s) > 0 {
		r, _ := utf8.DecodeRuneInString(s)
		if r != '。' && r != '.' && r != ' ' && r != '\t' {
			break
		}
		s = strings.TrimPrefix(s, string(r))
	}
	for len(s) > 0 {
		r, size := utf8.DecodeLastRuneInString(s)
		if r != '。' && r != '.' && r != ' ' && r != '\t' {
			break
		}
		s = s[:len(s)-size]
	}
	return s
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
