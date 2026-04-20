package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	NewAPIToolsURL    string
	NewAPIToolsAPIKey string
	PostgresDSN       string
	Port              string
	ExportStorageDir  string // 导出文件存储目录
}

func Load() *Config {
	// 尝试从 .env 文件加载环境变量（如果存在）
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables or defaults")
	}

	cfg := &Config{
		NewAPIToolsURL:    getEnv("NEWAPI_TOOLS_URL", "http://localhost:8081"),
		NewAPIToolsAPIKey: getEnv("SERVICE_API_KEY", ""),
		PostgresDSN:       buildPostgresDSN(),
		Port:              getEnv("PORT", "8080"),
		ExportStorageDir:  getEnv("EXPORT_STORAGE_DIR", "./exports"),
	}

	// 确保导出存储目录存在
	if err := os.MkdirAll(cfg.ExportStorageDir, 0755); err != nil {
		log.Printf("Warning: failed to create export storage directory: %v", err)
	}

	return cfg
}

// buildPostgresDSN 优先使用完整的 POSTGRES_DSN，否则从各独立变量拼接
func buildPostgresDSN() string {
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		return dsn
	}
	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")
	user := getEnv("POSTGRES_USER", "postgres")
	password := getEnv("POSTGRES_PASSWORD", "postgres")
	dbname := getEnv("POSTGRES_DB", "export_service")
	sslmode := getEnv("POSTGRES_SSLMODE", "disable")
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
