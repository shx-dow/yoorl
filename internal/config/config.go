package config

import (
	"os"
	"time"
)

type Config struct {
	Port        string
	BaseURL     string
	RedisAddr   string
	RedisPass   string
	CacheTTL    time.Duration
	RateLimit   int
	RateBurst   int
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func Load() *Config {
	return &Config{
		Port:      getEnv("PORT", "8080"),
		BaseURL:   getEnv("BASE_URL", "http://localhost:8080/"),
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass: getEnv("REDIS_PASSWORD", ""),
		CacheTTL:  6 * time.Hour,
		RateLimit: 100,
		RateBurst: 50,
	}
}
