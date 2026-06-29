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

type UserHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewUserHandler(db *gorm.DB, cfg *config.Config) *UserHandler {
	return &UserHandler{db: db, cfg: cfg}
}

func (h *UserHandler) List(c *gin.Context) {
	var list []models.User
	h.db.Order("id desc").Find(&list)
	utils.OK(c, list)
}

type userCreateReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
}

func (h *UserHandler) Create(c *gin.Context) {
	var req userCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, 400, "参数错误")
		return
	}
	var count int64
	h.db.Model(&models.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		utils.Fail(c, 1003, "账号已存在")
		return
	}
	u, err := services.EnsureUser(h.db, req.Username, req.Password, req.Nickname, req.Role)
	if err != nil {
		utils.Fail(c, 500, "创建失败")
		return
	}
	utils.OK(c, u)
}

func (h *UserHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var u models.User
	if err := h.db.First(&u, id).Error; err != nil {
		utils.Fail(c, 404, "不存在")
		return
	}
	var patch map[string]any
	c.ShouldBindJSON(&patch)
	// 防止密码明文落入
	if _, ok := patch["password"]; ok {
		delete(patch, "password")
	}
	if pwd, ok := patch["newPassword"].(string); ok && pwd != "" {
		hash, _ := utils.HashPassword(pwd)
		patch["password"] = hash
		delete(patch, "newPassword")
	}
	if r, ok := patch["role"].(string); ok && r != "" && r != "admin" && r != "employee" {
		delete(patch, "role")
	}
	h.db.Model(&u).Updates(patch)
	utils.OK(c, u)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id == 1 {
		utils.Fail(c, 1004, "默认管理员不可删除")
		return
	}
	h.db.Delete(&models.User{}, id)
	utils.OK(c, nil)
}

type OperationLogHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewOperationLogHandler(db *gorm.DB, cfg *config.Config) *OperationLogHandler {
	return &OperationLogHandler{db: db, cfg: cfg}
}

func (h *OperationLogHandler) List(c *gin.Context) {
	q := h.db.Model(&models.OperationLog{})
	if uid := c.Query("userId"); uid != "" {
		q = q.Where("user_id = ?", uid)
	}
	if action := c.Query("action"); action != "" {
		q = q.Where("action = ?", action)
	}
	if kw := c.Query("keyword"); kw != "" {
		q = q.Where("resource LIKE ? OR detail LIKE ? OR username LIKE ?", "%"+kw+"%", "%"+kw+"%", "%"+kw+"%")
	}
	var list []models.OperationLog
	q.Order("id desc").Limit(500).Find(&list)
	utils.OK(c, list)
}
