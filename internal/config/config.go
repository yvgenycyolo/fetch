package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Server settings
	ServerAddress string
	
	// Fetch settings
	FetchTimeout       time.Duration
	MaxRedirects       int
	MaxContentSize     int64
	
	// Rate limiting settings
	RateLimitRequests int
	RateLimitWindow   time.Duration
	RateLimitBurst    int
	
	// Cleanup/TTL settings
	ResultTTL          time.Duration
	CleanupInterval    time.Duration
	MaxResultsInMemory int
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		ServerAddress:      getEnv("SERVER_ADDRESS", ":8080"),
		FetchTimeout:       getDurationEnv("FETCH_TIMEOUT", 30*time.Second),
		MaxRedirects:       getIntEnv("MAX_REDIRECTS", 10),
		MaxContentSize:     getInt64Env("MAX_CONTENT_SIZE", 10*1024*1024), // 10MB
		RateLimitRequests:  getIntEnv("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:    getDurationEnv("RATE_LIMIT_WINDOW", 1*time.Minute),
		RateLimitBurst:     getIntEnv("RATE_LIMIT_BURST", 20),
		ResultTTL:          getDurationEnv("RESULT_TTL", 1*time.Hour),
		CleanupInterval:    getDurationEnv("CLEANUP_INTERVAL", 10*time.Minute),
		MaxResultsInMemory: getIntEnv("MAX_RESULTS_IN_MEMORY", 10000),
	}
}

// LogConfig logs the current configuration
func (c *Config) LogConfig() {
	log.Println("Configuration:")
	log.Printf("  Server Address: %s", c.ServerAddress)
	log.Printf("  Fetch Timeout: %v", c.FetchTimeout)
	log.Printf("  Max Redirects: %d", c.MaxRedirects)
	log.Printf("  Max Content Size: %d bytes (%.2f MB)", c.MaxContentSize, float64(c.MaxContentSize)/1024/1024)
	log.Printf("  Rate Limit: %d requests per %v (burst: %d)", c.RateLimitRequests, c.RateLimitWindow, c.RateLimitBurst)
	log.Printf("  Result TTL: %v", c.ResultTTL)
	log.Printf("  Cleanup Interval: %v", c.CleanupInterval)
	log.Printf("  Max Results in Memory: %d", c.MaxResultsInMemory)
}

// getEnv gets a string environment variable or returns default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnv gets an integer environment variable or returns default
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		log.Printf("Warning: Invalid integer value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}

// getInt64Env gets an int64 environment variable or returns default
func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if int64Value, err := strconv.ParseInt(value, 10, 64); err == nil {
			return int64Value
		}
		log.Printf("Warning: Invalid int64 value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}

// getDurationEnv gets a duration environment variable or returns default
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		log.Printf("Warning: Invalid duration value for %s: %s, using default: %v", key, value, defaultValue)
	}
	return defaultValue
}


