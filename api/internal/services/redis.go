package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func RedisConnect() (*redis.Client, error) {
	redisURL := strings.Trim(strings.TrimSpace(os.Getenv("REDIS_URL")), "\"'")
	if redisURL == "" {
		return nil, fmt.Errorf("REDIS_URL environment variable is not set")
	}

	var opt *redis.Options
	var err error

	if strings.HasPrefix(redisURL, "redis://") || strings.HasPrefix(redisURL, "rediss://") {
		opt, err = redis.ParseURL(redisURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse REDIS_URL: %w", err)
		}
	} else {
		opt = &redis.Options{
			Addr: redisURL,
			DB:   0,
		}
	}

	rdb := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis failed to connect: %w", err)
	}

	log.Println("Redis Connected...")
	return rdb, nil
}
