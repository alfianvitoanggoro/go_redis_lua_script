package cache

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"time"

	"grls/internal/config"

	"github.com/redis/go-redis/v9"
)

// NewAPI: client untuk jalur sinkron (gRPC → Lua/EVAL)
func NewAPI(ctx context.Context, cfg config.RedisConfig) (redis.UniversalClient, error) {
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	dbIndex, _ := strconv.Atoi(cfg.DB)

	// Pool sizing: sesuaikan beban (mis. 500 VU → mulai 300)
	poolSize := 300
	minIdle := 100

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       dbIndex,

		// Timeouts (sinkron path cepat)
		DialTimeout:  1 * time.Second,
		ReadTimeout:  400 * time.Millisecond,
		WriteTimeout: 400 * time.Millisecond,

		// Pooling
		PoolSize:     poolSize,
		MinIdleConns: minIdle,
		PoolTimeout:  300 * time.Millisecond,

		// v9 pengganti IdleTimeout
		ConnMaxIdleTime: 90 * time.Second,
		ConnMaxLifetime: 0,

		// Kurangi tail latency pada high contention
		PoolFIFO: true,

		// Retry kecil saja (hindari retry panjang di hot path)
		MaxRetries:      1,
		MinRetryBackoff: 50 * time.Millisecond,
		MaxRetryBackoff: 200 * time.Millisecond,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}
	return rdb, nil
}

// NewWorker: client khusus worker (XREADGROUP blocking)
func NewWorker(ctx context.Context, cfg config.RedisConfig, workerCount int) (redis.UniversalClient, error) {
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	dbIndex, _ := strconv.Atoi(cfg.DB)

	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}
	// 1 koneksi blocking per worker + ekstra untuk ack/xclaim, dll.
	poolSize := workerCount + 64
	minIdle := workerCount

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       dbIndex,

		// IMPORTANT: ReadTimeout harus >= Block (atau 0 = no timeout) untuk XREADGROUP
		DialTimeout: 1 * time.Second,
		ReadTimeout: 0, // no timeout: cocok untuk XREADGROUP Block N detik
		// Write untuk ACK/XCLAIM tetap singkat
		WriteTimeout: 1 * time.Second,

		PoolSize:        poolSize,
		MinIdleConns:    minIdle,
		PoolTimeout:     2 * time.Second,
		ConnMaxIdleTime: 120 * time.Second,
		ConnMaxLifetime: 0,
		PoolFIFO:        true,

		MaxRetries:      1,
		MinRetryBackoff: 100 * time.Millisecond,
		MaxRetryBackoff: 500 * time.Millisecond,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}
	return rdb, nil
}
