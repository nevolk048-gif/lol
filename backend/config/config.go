package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Security SecurityConfig
	Telegram TelegramConfig
}

type TelegramConfig struct {
	BotToken      string
	DisputeChatID string
}

type ServerConfig struct {
	Port        string
	Environment string
	BaseURL     string
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type SecurityConfig struct {
	EncryptionKey string
	RateLimitRPS  int
}

func Load() *Config {
	accessTTL, _ := strconv.Atoi(getEnv("JWT_ACCESS_TTL_MINUTES", "525600"))  // 1 year
	refreshTTL, _ := strconv.Atoi(getEnv("JWT_REFRESH_TTL_HOURS", "87600"))   // 10 years
	rateLimit, _ := strconv.Atoi(getEnv("RATE_LIMIT_RPS", "100"))

	return &Config{
		Server: ServerConfig{
			Port:        getEnv("SERVER_PORT", "8080"),
			Environment: getEnv("ENVIRONMENT", "development"),
			BaseURL:     getEnv("BASE_URL", "http://localhost:8080"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://paymentsgate:paymentsgate@localhost:5432/paymentsgate?sslmode=disable"),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379/0"),
		},
		JWT: JWTConfig{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", "dev-access-secret-change-in-production"),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", "dev-refresh-secret-change-in-production"),
			AccessTTL:     time.Duration(accessTTL) * time.Minute,
			RefreshTTL:    time.Duration(refreshTTL) * time.Hour,
		},
		Security: SecurityConfig{
			EncryptionKey: getEnv("ENCRYPTION_KEY", "32-byte-dev-encryption-key!!"),
			RateLimitRPS:  rateLimit,
		},
		Telegram: TelegramConfig{
			BotToken:      getEnv("TELEGRAM_BOT_TOKEN", ""),
			DisputeChatID: getEnv("TELEGRAM_DISPUTE_CHAT_ID", ""),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
