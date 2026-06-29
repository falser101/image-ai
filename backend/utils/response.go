package utils

import "github.com/gin-gonic/gin"

func OK(c *gin.Context, data any) {
	c.JSON(200, gin.H{"code": 0, "data": data})
}

func Fail(c *gin.Context, code int, msg string) {
	c.JSON(200, gin.H{"code": code, "message": msg})
}
