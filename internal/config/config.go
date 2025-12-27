package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPath         string
	Port           int
	SessionSecret  string
	SMTPConfig     SMTPConfig
	TelegramConfig TelegramConfig
	CheckInterval  time.Duration
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type TelegramConfig struct {
	BotToken string
}

func (c *Config) ServerAddr() string {
	return fmt.Sprintf(":%d", c.Port)
}

func Load() (*Config, error) {
	godotenv.Load()

	cfg := &Config{
		DBPath:        getEnv("DB_PATH", "data/domain_minder.db"),
		Port:          getEnvInt("PORT", 9000),
		SessionSecret: getEnv("SESSION_SECRET", "change-this-in-production"),
		CheckInterval: getEnvDuration("CHECK_INTERVAL", 6*time.Hour),
		SMTPConfig: SMTPConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     getEnvInt("SMTP_PORT", 587),
			Username: getEnv("SMTP_USER", ""),
			Password: getEnv("SMTP_PASS", ""),
			From:     getEnv("SMTP_FROM", "noreply@domain-minder.local"),
		},
		TelegramConfig: TelegramConfig{
			BotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultValue
}
