package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort           = "8080"
	defaultOrigins        = "http://localhost:5173"
	defaultBucket         = "wlt-public-sandbox"
	defaultRequestTimeout = 15 * time.Second
	defaultWorkerCount    = 8
	defaultPageSize       = 100
	defaultPrefetchPages  = 1
	minPageSize           = 25
	maxPageSize           = 500
)

// Config holds runtime configuration for the backend server.
type Config struct {
	Port            string
	AllowedOrigins  []string
	Bucket          string
	RequestTimeout  time.Duration
	WorkerCount     int
	DefaultPageSize int
	MinPageSize     int
	MaxPageSize     int
	PrefetchPages   int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	cfg := Config{
		Port:            getEnv("PORT", defaultPort),
		AllowedOrigins:  splitAndTrim(getEnv("ALLOWED_ORIGINS", defaultOrigins)),
		Bucket:          getEnv("GCS_BUCKET", defaultBucket),
		RequestTimeout:  getDurationEnv("REQUEST_TIMEOUT", defaultRequestTimeout),
		WorkerCount:     getIntEnv("WORKER_COUNT", defaultWorkerCount),
		DefaultPageSize: getIntEnv("DEFAULT_PAGE_SIZE", defaultPageSize),
		MinPageSize:     getIntEnv("MIN_PAGE_SIZE", minPageSize),
		MaxPageSize:     getIntEnv("MAX_PAGE_SIZE", maxPageSize),
		PrefetchPages:   getIntEnv("PREFETCH_PAGES", defaultPrefetchPages),
	}

	if cfg.MinPageSize < 1 {
		cfg.MinPageSize = minPageSize
	}
	if cfg.MaxPageSize < cfg.MinPageSize {
		cfg.MaxPageSize = maxPageSize
	}
	if cfg.DefaultPageSize < cfg.MinPageSize || cfg.DefaultPageSize > cfg.MaxPageSize {
		cfg.DefaultPageSize = defaultPageSize
	}
	if cfg.WorkerCount < 1 {
		cfg.WorkerCount = defaultWorkerCount
	}
	if cfg.PrefetchPages < 0 {
		cfg.PrefetchPages = 0
	}

	return cfg
}

func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	var cleaned []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return cleaned
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			return parsed
		}
	}
	return fallback
}
