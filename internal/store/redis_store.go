package store

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed lua/deposit.lua
var luaDeposit string

const streamWallet = "stream:wallet"

type RedisWalletStore struct {
	rdb        redis.UniversalClient
	scrDeposit *redis.Script
}

func NewRedisWalletStore(rdb redis.UniversalClient) *RedisWalletStore {
	s := &RedisWalletStore{
		rdb:        rdb,
		scrDeposit: redis.NewScript(luaDeposit),
	}

	// Preload SHA agar call pertama tidak kena EVAL penuh / NOSCRIPT
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.scrDeposit.Load(ctx, rdb).Err() // abaikan error; Run() tetap fallback
	}()

	return s
}

func keyBalance(userID, currency string) string {
	return fmt.Sprintf("balance:{%s}:%s", userID, strings.ToUpper(currency))
}
func keyTx(userID string) string { return fmt.Sprintf("tx:{%s}", userID) }
func nowMillis() string          { return strconv.FormatInt(time.Now().UnixMilli(), 10) }

type TxResult struct {
	Code    int64
	Applied bool
	Balance int64 // minor units (integer)
}

func (s *RedisWalletStore) Deposit(ctx context.Context, userID, currency, txID string, amount int64, meta map[string]any) (TxResult, error) {
	cur := strings.ToUpper(currency)
	metaJSON, _ := json.Marshal(meta)

	keys := []string{
		keyBalance(userID, cur), // KEYS[1]
		keyTx(userID),           // KEYS[2]
		streamWallet,            // KEYS[3]
	}
	// ARGV: txId, amount, ts, metaJSON, userID, currency
	args := []any{txID, amount, nowMillis(), string(metaJSON), userID, cur}

	raw, err := s.scrDeposit.Run(ctx, s.rdb, keys, args...).Result()
	if err != nil {
		return TxResult{}, err
	}
	arr := raw.([]interface{})
	code := arr[0].(int64)
	bal, _ := strconv.ParseInt(arr[1].(string), 10, 64)
	return TxResult{Code: code, Applied: code == 1, Balance: bal}, nil
}
