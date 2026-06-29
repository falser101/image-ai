package main

import (
	"log"

	"github.com/image-ai/backend/config"
	"github.com/image-ai/backend/database"
	"github.com/image-ai/backend/handlers"
	"github.com/image-ai/backend/middleware"
	"github.com/image-ai/backend/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	db := database.Init(cfg)

	// 初始化默认管理员
	if err := services.EnsureDefaultAdmin(db, cfg); err != nil {
		log.Fatalf("init default admin failed: %v", err)
	}

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * 3600,
	}))

	// 静态资源（本地存储）
	r.Static("/uploads", cfg.UploadDir)

	// 健康检查
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	auth := handlers.NewAuthHandler(db, cfg)
	productH := handlers.NewProductHandler(db, cfg)
	pointH := handlers.NewSellingPointHandler(db, cfg)
	imageH := handlers.NewImageHandler(db, cfg)
	galleryH := handlers.NewGalleryHandler(db, cfg)
	modelH := handlers.NewModelConfigHandler(db, cfg)
	ossH := handlers.NewOssConfigHandler(db, cfg)
	styleH := handlers.NewStylePresetHandler(db, cfg)
	userH := handlers.NewUserHandler(db, cfg)
	logH := handlers.NewOperationLogHandler(db, cfg)
	aiH := handlers.NewAIHandler(db, cfg)
	catH := handlers.NewProviderCatalogHandler(db, cfg)

	api := r.Group("/api")
	{
		api.POST("/auth/login", auth.Login)
		api.POST("/auth/register", auth.Register)
		api.GET("/providers", catH.List)
	}

	authed := api.Group("")
	authed.Use(middleware.JWTAuth(cfg.JWTSecret))
	authed.Use(middleware.OperationLog(db))
	{
		authed.GET("/auth/me", auth.Me)
		authed.POST("/auth/password", auth.ChangePassword)

		// 产品
		authed.POST("/products", productH.Create)
		authed.GET("/products", productH.List)
		authed.GET("/products/:id", productH.Get)
		authed.DELETE("/products/:id", productH.Delete)
		authed.POST("/products/:id/source-images", productH.UploadSourceImage)
		authed.POST("/products/:id/selling-points", pointH.CreateForProduct)
		authed.GET("/products/:id/selling-points", pointH.ListByProduct)

		// 卖点
		authed.GET("/selling-points", pointH.List)
		authed.GET("/selling-points/:id", pointH.Get)
		authed.DELETE("/selling-points/:id", pointH.Delete)

		// AI 分析与生图
		authed.POST("/ai/analyze", aiH.Analyze)
		authed.POST("/ai/generate", aiH.Generate)
		authed.GET("/ai/tasks/:id", aiH.TaskStatus)

		// 图片（原图与生成图统一管理）
		authed.GET("/images", imageH.List)
		authed.GET("/images/:id", imageH.Get)
		authed.DELETE("/images/:id", imageH.Delete)
		authed.GET("/images/:id/file", imageH.ServeFile)

		// 图库（生成结果）
		authed.GET("/gallery", galleryH.List)
		authed.GET("/gallery/:id", galleryH.Get)
		authed.DELETE("/gallery/:id", galleryH.Delete)
		authed.GET("/gallery/:id/file", galleryH.ServeFile)

		// 模型配置（仅管理员）
		admin := authed.Group("")
		admin.Use(middleware.RequireAdmin())
		{
			admin.GET("/model-configs", modelH.List)
			admin.POST("/model-configs", modelH.Create)
			admin.PUT("/model-configs/:id", modelH.Update)
			admin.DELETE("/model-configs/:id", modelH.Delete)

			admin.POST("/providers/fetch-models", catH.FetchModels)

			admin.GET("/model-presets/:provider/:type", modelH.Presets)

			admin.GET("/oss-config", ossH.Get)
			admin.PUT("/oss-config", ossH.Update)

			admin.GET("/style-presets", styleH.List)
			admin.POST("/style-presets", styleH.Create)
			admin.PUT("/style-presets/:id", styleH.Update)
			admin.DELETE("/style-presets/:id", styleH.Delete)

			admin.GET("/users", userH.List)
			admin.POST("/users", userH.Create)
			admin.PUT("/users/:id", userH.Update)
			admin.DELETE("/users/:id", userH.Delete)

			admin.GET("/operation-logs", logH.List)
		}
	}

	addr := ":" + cfg.Port
	log.Printf("server starting on %s, upload dir=%s, db=%s", addr, cfg.UploadDir, cfg.DBPath)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
