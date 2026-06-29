package handlers

import (
	"time"

	"github.com/image-ai/backend/config"
	"github.com/image-ai/backend/models"
	"github.com/image-ai/backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg}
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, 400, "参数错误")
		return
	}
	var u models.User
	if err := h.db.Where("username = ?", req.Username).First(&u).Error; err != nil {
		utils.Fail(c, 1001, "账号或密码错误")
		return
	}
	if !utils.CheckPassword(u.Password, req.Password) {
		utils.Fail(c, 1001, "账号或密码错误")
		return
	}
	if u.Status != "active" {
		utils.Fail(c, 1002, "账号已停用")
		return
	}
	tok, _ := utils.GenerateToken(h.cfg.JWTSecret, u.ID, u.Username, u.Role, 24*7*time.Hour)
	utils.OK(c, gin.H{
		"token": tok,
		"user": gin.H{
			"id":       u.ID,
			"username": u.Username,
			"nickname": u.Nickname,
			"role":     u.Role,
		},
	})
	h.db.Create(&models.OperationLog{
		UserID: u.ID, Username: u.Username, Action: "LOGIN", Resource: "/api/auth/login", IP: c.ClientIP(),
	})
}

type registerReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	Nickname string `json:"nickname"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
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
	hash, _ := utils.HashPassword(req.Password)
	u := models.User{Username: req.Username, Password: hash, Nickname: req.Nickname, Role: "employee", Status: "active"}
	if err := h.db.Create(&u).Error; err != nil {
		utils.Fail(c, 500, "注册失败")
		return
	}
	utils.OK(c, gin.H{"id": u.ID, "username": u.Username})
}

func (h *AuthHandler) Me(c *gin.Context) {
	uid, _ := c.Get("userId")
	var u models.User
	if err := h.db.First(&u, uid).Error; err != nil {
		utils.Fail(c, 404, "用户不存在")
		return
	}
	utils.OK(c, gin.H{
		"id": u.ID, "username": u.Username, "nickname": u.Nickname, "role": u.Role, "status": u.Status,
	})
}

type changePwdReq struct {
	Old string `json:"old" binding:"required"`
	New string `json:"new" binding:"required,min=6"`
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePwdReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, 400, "参数错误")
		return
	}
	uid, _ := c.Get("userId")
	var u models.User
	if err := h.db.First(&u, uid).Error; err != nil {
		utils.Fail(c, 404, "用户不存在")
		return
	}
	if !utils.CheckPassword(u.Password, req.Old) {
		utils.Fail(c, 1001, "原密码错误")
		return
	}
	hash, _ := utils.HashPassword(req.New)
	u.Password = hash
	h.db.Save(&u)
	utils.OK(c, nil)
}
