package store

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed lua/deposit.lua
var luaDeposit string

type RedisWalletStore struct {
	rdb        redis.UniversalClient
	scrDeposit *redis.Script
}

func NewRedisWalletStore(rdb redis.UniversalClient) *RedisWalletStore {
	return &RedisWalletStore{
		rdb:        rdb,
		scrDeposit: redis.NewScript(luaDeposit),
	}
}

func keyBalance(userID, currency string) string {
	return fmt.Sprintf("balance:{%s}:%s", userID, currency)
}
func keyTx(userID string) string     { return fmt.Sprintf("tx:{%s}", userID) }
func keyStream(userID string) string { return fmt.Sprintf("stream:balance:{%s}", userID) }
func nowMillis() string              { return strconv.FormatInt(time.Now().UnixMilli(), 10) }

type TxResult struct {
	Code    int64
	Applied bool
	Balance int64 // minor units
}

func (s *RedisWalletStore) Deposit(ctx context.Context, userID, currency, txID string, amount int64, meta map[string]any) (TxResult, error) {
	metaJSON, _ := json.Marshal(meta)
	keys := []string{keyBalance(userID, currency), keyTx(userID), keyStream(userID)}
	args := []any{txID, amount, nowMillis(), string(metaJSON)}

	raw, err := s.scrDeposit.Run(ctx, s.rdb, keys, args...).Result()
	if err != nil {
		return TxResult{}, err
	}
	arr := raw.([]interface{})
	code := arr[0].(int64)
	bal, _ := strconv.ParseInt(arr[1].(string), 10, 64)
	return TxResult{Code: code, Applied: code == 1, Balance: bal}, nil
}
