package cache

import (
	"context"
	"fmt"
	"grls/internal/config"
	"strconv"

	"github.com/redis/go-redis/v9"
)

func New(ctx context.Context, redisConfig config.RedisConfig) (redis.UniversalClient, error) {
	redisURL := fmt.Sprintf("%s:%s", redisConfig.Host, redisConfig.Port)

	dbIndex, err := strconv.Atoi(redisConfig.DB)

	if err != nil {
		dbIndex = 0 // fallback ke 0 jika gagal convert
	}

	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: redisConfig.Password,
		DB:       dbIndex,
	})

	// Test connection
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	return rdb, nil
}
