package controller_test

import (
	"export-service/config"
	"export-service/controller"
	"export-service/model"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared",
		strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test sqlite: %v", err)
	}
	model.DB = db

	if err := db.AutoMigrate(&model.FeishuTokenMapping{}, &model.ExportLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
}

// newTestRouter 直接构建路由，不依赖 router 包（避免循环导入）
func newTestRouter(t *testing.T, newAPIURL string) *gin.Engine {
	t.Helper()
	cfg := &config.Config{
		NewAPIToolsURL: newAPIURL,
		PostgresDSN:    "",
		Port:           "8080",
	}

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

	mappingCtrl := controller.NewMappingController()
	exportCtrl := controller.NewExportController(cfg)

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		mappings := api.Group("/mappings")
		{
			mappings.GET("", mappingCtrl.List)
			mappings.POST("", mappingCtrl.Create)
			mappings.PUT("/:id", mappingCtrl.Update)
			mappings.DELETE("/:id", mappingCtrl.Delete)
			mappings.DELETE("", mappingCtrl.Delete)
			mappings.POST("/batch-delete", mappingCtrl.BatchDelete)
			mappings.POST("/import", mappingCtrl.Import)
		}
		exports := api.Group("/export")
		{
			exports.POST("/preview", exportCtrl.Preview)
			exports.POST("/download", exportCtrl.Download)
		}
		api.GET("/history", exportCtrl.GetExportLogs)
	}

	return r
}
