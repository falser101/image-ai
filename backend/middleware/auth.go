package middleware

import (
	"strings"

	"github.com/image-ai/backend/utils"

	"github.com/gin-gonic/gin"
)

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Fail(c, 401, "未登录")
			c.Abort()
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.Fail(c, 401, "无效令牌")
			c.Abort()
			return
		}
		claims, err := utils.ParseToken(secret, parts[1])
		if err != nil {
			utils.Fail(c, 401, "令牌过期或无效")
			c.Abort()
			return
		}
		c.Set("userId", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			utils.Fail(c, 403, "需要管理员权限")
			c.Abort()
			return
		}
		c.Next()
	}
}
