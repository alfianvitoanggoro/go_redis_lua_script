package store

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed lua/enqueue_and_try_acquire.lua
var luaEnqueue string

//go:embed lua/try_promote_if_head.lua
var luaPromote string

//go:embed lua/release_and_promote.lua
var luaRelease string

//go:embed lua/force_release.lua
var luaForce string

type FIFOLock struct {
	rdb      redis.UniversalClient
	ttl      time.Duration
	scrEnq   *redis.Script
	scrProm  *redis.Script
	scrRel   *redis.Script
	scrForce *redis.Script
}

func NewFIFOLock(rdb redis.UniversalClient, ttl time.Duration) *FIFOLock {
	l := &FIFOLock{
		rdb:      rdb,
		ttl:      ttl,
		scrEnq:   redis.NewScript(luaEnqueue),
		scrProm:  redis.NewScript(luaPromote),
		scrRel:   redis.NewScript(luaRelease),
		scrForce: redis.NewScript(luaForce),
	}
	// preload scripts (best-effort)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = l.scrEnq.Load(ctx, rdb).Err()
		_ = l.scrProm.Load(ctx, rdb).Err()
		_ = l.scrRel.Load(ctx, rdb).Err()
		_ = l.scrForce.Load(ctx, rdb).Err()
	}()
	return l
}

func qKey(user string) string   { return fmt.Sprintf("q:{%s}", user) }
func ownKey(user string) string { return fmt.Sprintf("own:{%s}", user) }

// Enqueue + coba acquire jika head & owner kosong
func (l *FIFOLock) EnqueueAndTryAcquire(ctx context.Context, user, reqID string) (bool, error) {
	keys := []string{qKey(user), ownKey(user)}
	args := []any{reqID, int64(l.ttl / time.Millisecond)}
	res, err := l.scrEnq.Run(ctx, l.rdb, keys, args...).Result()
	if err != nil {
		return false, err
	}
	arr := res.([]interface{})
	code := arr[0].(int64) // 1=acquired, 0=queued
	return code == 1, nil
}

// Coba promote (atau perpanjang lease bila sudah owner)
func (l *FIFOLock) TryPromoteIfHead(ctx context.Context, user, reqID string) (bool, error) {
	keys := []string{qKey(user), ownKey(user)}
	args := []any{reqID, int64(l.ttl / time.Millisecond)}
	res, err := l.scrProm.Run(ctx, l.rdb, keys, args...).Result()
	if err != nil {
		return false, err
	}
	arr := res.([]interface{})
	code := arr[0].(int64) // 1=owner/acquired, 0=waiting
	return code == 1, nil
}

// Lepas slot & promosikan next (FIFO)
func (l *FIFOLock) ReleaseAndPromote(ctx context.Context, user, reqID string) error {
	keys := []string{qKey(user), ownKey(user)}
	args := []any{reqID, int64(l.ttl / time.Millisecond)}
	_, err := l.scrRel.Run(ctx, l.rdb, keys, args...).Result()
	return err
}

// Recovery (opsional): paksa hapus owner jika macet
func (l *FIFOLock) ForceRelease(ctx context.Context, user, expectedOwner string) error {
	keys := []string{qKey(user), ownKey(user)}
	args := []any{expectedOwner}
	_, err := l.scrForce.Run(ctx, l.rdb, keys, args...).Result()
	return err
}

// Helper: tunggu sampai dapat giliran (polling ringan)
func (l *FIFOLock) WaitUntilAcquired(ctx context.Context, user, reqID string, maxWait, pollInterval time.Duration) (bool, error) {
	deadline := time.Now().Add(maxWait)
	for {
		ok, err := l.TryPromoteIfHead(ctx, user, reqID)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
		if time.Now().After(deadline) {
			return false, nil // timeout
		}
		time.Sleep(pollInterval)
	}
}
