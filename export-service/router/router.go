package router

import (
	"export-service/config"
	"export-service/controller"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// SetupRouterWithoutStatic 创建不带静态文件服务的路由（用于测试）
func SetupRouterWithoutStatic(cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	mappingController := controller.NewMappingController()
	exportController := controller.NewExportController(cfg)

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		mappings := api.Group("/mappings")
		{
			mappings.GET("", mappingController.List)
			mappings.POST("", mappingController.Create)
			mappings.PUT("/:id", mappingController.Update)
			mappings.DELETE("/:id", mappingController.Delete)
			mappings.DELETE("", mappingController.Delete)
			mappings.POST("/batch-delete", mappingController.BatchDelete)
			mappings.POST("/import", mappingController.Import)
		}

		exports := api.Group("/export")
		{
			exports.POST("/preview", exportController.Preview)
			exports.POST("/download", exportController.Download)
			exports.GET("/download-file/:id", exportController.DownloadFile)
		}

		api.GET("/history", exportController.GetExportLogs)
	}

	return r
}

// SetupRouter 设置路由
func SetupRouter(cfg *config.Config) *gin.Engine {
	r := gin.Default()

	// CORS中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 初始化控制器
	mappingController := controller.NewMappingController()
	exportController := controller.NewExportController(cfg)

	// 健康检查
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API路由组
	api := r.Group("/api")
	{
		// 映射管理
		mappings := api.Group("/mappings")
		{
			mappings.GET("", mappingController.List)
			mappings.POST("", mappingController.Create)
			mappings.PUT("/:id", mappingController.Update)
			mappings.DELETE("/:id", mappingController.Delete)
			mappings.DELETE("", mappingController.Delete)       // 支持批量删除 ?label=xxx
			mappings.POST("/batch-delete", mappingController.BatchDelete) // 批量删除
			mappings.POST("/import", mappingController.Import)  // CSV文件导入
		}

		// 导出功能
		exports := api.Group("/export")
		{
			exports.POST("/preview", exportController.Preview)
			exports.POST("/download", exportController.Download)
			exports.GET("/download-file/:id", exportController.DownloadFile)
		}

		// 导出日志
		api.GET("/history", exportController.GetExportLogs)
	}

	// 静态文件服务（检查本地 dist 目录）
	setupStaticFiles(r)

	return r
}

// setupStaticFiles 设置静态文件服务
// 运行时检查 web/dist 目录是否存在，不存在则跳过
func setupStaticFiles(r *gin.Engine) {
	// 可能的静态文件路径（按优先级）
	possiblePaths := []string{
		"web/dist",           // 开发环境
		"./web/dist",         // 相对路径
		"/app/web/dist",      // Docker 环境
	}

	var distPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			// 检查 index.html 是否存在
			if _, err := os.Stat(filepath.Join(path, "index.html")); err == nil {
				distPath = path
				break
			}
		}
	}

	if distPath == "" {
		// 静态文件目录不存在，跳过
		return
	}

	// 提供静态文件服务
	r.Static("/assets", filepath.Join(distPath, "assets"))
	r.StaticFile("/favicon.ico", filepath.Join(distPath, "favicon.ico"))

	// 所有其他请求返回 index.html（支持前端路由）
	r.NoRoute(func(c *gin.Context) {
		c.File(filepath.Join(distPath, "index.html"))
	})
}
