package config

import (
	"os"
	"time"
)

type Config struct {
	DatabaseURL     string
	SecretKey       string
	DiffServiceURL  string
	LogLevel        string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func Load() *Config {
	return &Config{
		DatabaseURL:     getEnv("DATABASE_URL", "./data/blackbox.db"),
		SecretKey:       mustGetEnv("SECRET_KEY"),
		DiffServiceURL:  getEnv("DIFF_SERVICE_URL", "http://diff-service:5001"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func mustGetEnv(key string) string {
	v := getEnv(key, "")
	if v == "" {
		panic(key + " environment variable is required")
	}
	return v
}
