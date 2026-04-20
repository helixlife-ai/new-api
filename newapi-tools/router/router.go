package router

import (
	"net/http"

	"newapi-tools/config"
	"newapi-tools/controller"

	"github.com/gin-gonic/gin"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Service-Key")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 服务间鉴权中间件：仅当 SERVICE_API_KEY 已配置时生效
	serviceAuth := func(c *gin.Context) {
		if cfg.ServiceAPIKey == "" {
			c.Next()
			return
		}
		if c.GetHeader("X-Service-Key") != cfg.ServiceAPIKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}

	logController := controller.NewLogController()

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/api/logs", serviceAuth, logController.QueryLogs)
	r.GET("/api/logs/export", serviceAuth, logController.ExportLogs)

	return r
}
