package main

import (
	"log"

	"newapi-tools/config"
	"newapi-tools/model"
	"newapi-tools/router"
)

func main() {
	cfg := config.Load()

	if err := model.InitDB(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	r := router.SetupRouter()

	log.Printf("newapi-tools starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
