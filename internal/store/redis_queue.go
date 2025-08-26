package store

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed lua/enqueue_and_try_acquire.lua
var luaEnqueue string

//go:embed lua/release_and_promote.lua
var luaRelease string

type RedisQueue struct {
	rdb            redis.UniversalClient
	scrEnqueue     *redis.Script
	scrRelease     *redis.Script
	ReadyKey       string // e.g. "ready:wallet"
	KeyQueuePrefix string // e.g. "q"
	KeyLockPrefix  string // e.g. "lock"
}

func NewRedisQueue(rdb redis.UniversalClient) *RedisQueue {
	q := &RedisQueue{
		rdb:            rdb,
		scrEnqueue:     redis.NewScript(luaEnqueue),
		scrRelease:     redis.NewScript(luaRelease),
		ReadyKey:       "ready:wallet",
		KeyQueuePrefix: "q",
		KeyLockPrefix:  "lock",
	}
	// Preload scripts (best effort)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = q.scrEnqueue.Load(ctx, rdb).Err()
		_ = q.scrRelease.Load(ctx, rdb).Err()
	}()
	return q
}

func (q *RedisQueue) keyQueue(user string) string {
	return fmt.Sprintf("%s:{%s}", q.KeyQueuePrefix, user)
}
func (q *RedisQueue) keyLock(user string) string {
	return fmt.Sprintf("%s:{%s}", q.KeyLockPrefix, user)
}

type DepositPayload struct {
	UserID   string `json:"user_id"`
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
	TxID     string `json:"tx_id"`
}

// EnqueueDeposit: push payload ke q:{user}; jika acquire head → dorong ke ready
func (q *RedisQueue) EnqueueDeposit(ctx context.Context, p DepositPayload) (acquired bool, err error) {
	b, _ := json.Marshal(p)
	keys := []string{q.keyQueue(p.UserID), q.keyLock(p.UserID), q.ReadyKey}
	res, err := q.scrEnqueue.Run(ctx, q.rdb, keys, string(b)).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// ReleaseAndPromote: dipanggil setelah DB sukses
func (q *RedisQueue) ReleaseAndPromote(ctx context.Context, user string) (int64, error) {
	keys := []string{q.keyQueue(user), q.keyLock(user), q.ReadyKey}
	return q.scrRelease.Run(ctx, q.rdb, keys).Int64()
}

// ReadyKeyName: expose nama ready list (untuk BRPOP)
func (q *RedisQueue) ReadyKeyName() string { return q.ReadyKey }

// QueueKeyForUser: expose nama q:{user} (untuk LINDEX head)
func (q *RedisQueue) QueueKeyForUser(user string) string { return q.keyQueue(user) }
