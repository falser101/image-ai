package middleware

import (
	"time"

	"github.com/image-ai/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// OperationLog 操作日志中间件：把请求落到日志表
func OperationLog(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		// 跳过GET请求和登录、健康检查
		if c.Request.Method == "GET" || c.Request.Method == "OPTIONS" {
			return
		}
		if c.FullPath() == "/api/auth/login" || c.FullPath() == "/api/health" {
			return
		}
		uid, ok := c.Get("userId")
		uname, _ := c.Get("username")
		if !ok {
			return
		}
		dur := time.Since(start)
		detail := c.Request.Method + " " + c.Request.URL.Path + " (" + dur.String() + ")"
		db.Create(&models.OperationLog{
			UserID:    uid.(uint),
			Username:  asString(uname),
			Action:    c.Request.Method,
			Resource:  c.FullPath(),
			Detail:    detail,
			IP:        c.ClientIP(),
		})
	}
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
