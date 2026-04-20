package main

import (
	"log"

	"newapi-tools/config"
	"newapi-tools/model"
	"newapi-tools/router"
)

func main() {
	cfg := config.Load()

	if cfg.ServiceAPIKey == "" {
		log.Println("WARNING: SERVICE_API_KEY is not set, /api/logs endpoints are unauthenticated")
	}

	if err := model.InitDB(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	r := router.SetupRouter(cfg)

	log.Printf("newapi-tools starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
