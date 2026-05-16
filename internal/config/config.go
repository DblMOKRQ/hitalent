package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Config struct {
	HTTPAddr        string
	LogLevel        string
	DBUser          string
	DBPassword      string
	DBHost          string
	DBPort          string
	DBName          string
	SSLMode         string
	DBMigrationPath string
}

func MustLoad() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment variables")
	}

	return &Config{
		HTTPAddr:        getEnv("HTTP_ADDR", ":8080"),
		LogLevel:        getEnv("LOG_LEVEL", "debug"),
		DBUser:          getEnv("DB_USER", "postgres"),
		DBPassword:      getEnv("DB_PASSWORD", "123"),
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBName:          getEnv("DB_NAME", "hitalent"),
		DBMigrationPath: getEnv("DB_MIGRATION_PATH", "./migrations"),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
	}
}

// getEnv возвращает строку или значение по умолчанию
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultVal
}
