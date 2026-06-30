package middleware

import (
	"strings"
	"time"

	"github.com/image-ai/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// OperationLog 操作日志中间件：把请求落到日志表。
//
// Action/Resource 写成"动作·资源"组合，前端一个表格能一眼看出对什么做了什么：
//   Action   = HTTP method   （POST/PUT/DELETE/PATCH）
//   Resource = Gin 路由模板  （如 /api/products/:id）
//   ResourceID = 实际路径里的 :id 值
//   Detail   = "POST /api/products/13 (12ms)"，供搜索
//
// AI 调用本身额外落一行专门日志（resource=model action=ai.analyze/ai.generate，tokens 字段填充）
// 由 handlers 在调 AI 成功/失败后写入，不走本中间件。
func OperationLog(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		// 跳过GET请求和登录、健康检查
		if c.Request.Method == "GET" || c.Request.Method == "OPTIONS" {
			return
		}
		fp := c.FullPath()
		if fp == "/api/auth/login" || fp == "/api/health" {
			return
		}
		uid, ok := c.Get("userId")
		uname, _ := c.Get("username")
		if !ok {
			return
		}
		dur := time.Since(start)
		// ResourceID 尝试从实际 URL 里掏出 :id 段
		resourceID := extractResourceID(fp, c.Request.URL.Path)
		detail := c.Request.Method + " " + c.Request.URL.Path + " (" + dur.String() + ")"
		db.Create(&models.OperationLog{
			UserID:     uid.(uint),
			Username:   asString(uname),
			Action:     c.Request.Method,
			Resource:   fp,
			ResourceID: resourceID,
			Detail:     detail,
			IP:         c.ClientIP(),
		})
	}
}

// extractResourceID 从实际 URL Path 里抠出 FullPath 中 :id 段对应的值。
// 例：FullPath="/api/products/:id", Path="/api/products/13" → "13"
// 例：FullPath="/api/ai/analyze", Path="/api/ai/analyze" → ""
func extractResourceID(fullPath, realPath string) string {
	if !strings.Contains(fullPath, ":") {
		return ""
	}
	fpSegs := strings.Split(strings.TrimPrefix(fullPath, "/"), "/")
	rpSegs := strings.Split(strings.TrimPrefix(realPath, "/"), "/")
	if len(fpSegs) != len(rpSegs) {
		return ""
	}
	for i, s := range fpSegs {
		if strings.HasPrefix(s, ":") {
			return rpSegs[i]
		}
	}
	return ""
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
