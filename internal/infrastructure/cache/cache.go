package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"grls/internal/config"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis(ctx context.Context, cfg config.RedisConfig) (redis.UniversalClient, error) {
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)

	dbIndex, err := strconv.Atoi(cfg.DB)
	if err != nil {
		dbIndex = 0
	}

	opts := &redis.Options{
		Addr:            addr,
		Password:        cfg.Password,
		DB:              dbIndex,
		DialTimeout:     1 * time.Second,
		ReadTimeout:     400 * time.Millisecond,
		WriteTimeout:    400 * time.Millisecond,
		PoolSize:        300,
		MinIdleConns:    100,
		PoolTimeout:     750 * time.Millisecond,
		ConnMaxIdleTime: 90 * time.Second,
		ConnMaxLifetime: 0,
		PoolFIFO:        true,
		MaxRetries:      0,
		MinRetryBackoff: 50 * time.Millisecond,
		MaxRetryBackoff: 200 * time.Millisecond,

		OnConnect: func(ctx context.Context, cn *redis.Conn) error {
			// Biar gampang di-trace di Redis: CLIENT LIST/INFO
			_ = cn.ClientSetName(ctx, "grls").Err()
			return nil
		},
	}

	rdb := redis.NewClient(opts)

	// health check
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}

	// pre-warm pool (best effort)
	warm := min(opts.MinIdleConns, 64)
	for i := 0; i < warm; i++ {
		go func() { _ = rdb.Ping(ctx).Err() }()
	}

	return rdb, nil
}
