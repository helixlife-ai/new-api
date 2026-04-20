package main

import (
	"export-service/config"
	"export-service/model"
	"export-service/router"
	"log"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库
	if err := model.InitDB(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 设置路由
	r := router.SetupRouter(cfg)

	// 启动服务
	log.Printf("Server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
