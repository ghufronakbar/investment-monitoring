// config/redis.go
package config

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis(cfg *Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.REDIS_HOST, cfg.REDIS_PORT),
		Password: cfg.REDIS_PASS,
		DB:       cfg.REDIS_DB,
	})

	// Test Ping
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return rdb, nil
}
