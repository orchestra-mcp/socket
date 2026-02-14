package bridge

import (
	"os"
	"strconv"
)

// RedisConfig holds connection settings for the Redis pub/sub bridge.
type RedisConfig struct {
	Addr     string // Redis address, default "localhost:6379"
	Password string // Redis password, default ""
	DB       int    // Redis database number, default 0
	Prefix   string // Channel prefix, default "orchestra:ws:"
}

// DefaultRedisConfig returns a RedisConfig with sensible defaults.
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addr:   "localhost:6379",
		Prefix: "orchestra:ws:",
	}
}

// RedisConfigFromEnv loads Redis configuration from environment variables.
// Falls back to defaults for any missing values.
func RedisConfigFromEnv() *RedisConfig {
	cfg := DefaultRedisConfig()

	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		cfg.Addr = addr
	}
	if pw := os.Getenv("REDIS_PASSWORD"); pw != "" {
		cfg.Password = pw
	}
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if db, err := strconv.Atoi(dbStr); err == nil {
			cfg.DB = db
		}
	}
	if prefix := os.Getenv("REDIS_WS_PREFIX"); prefix != "" {
		cfg.Prefix = prefix
	}
	return cfg
}
