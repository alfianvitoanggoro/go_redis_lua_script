package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"grls/internal/infrastructure/repository"
	"grls/pkg/logger"

	"github.com/redis/go-redis/v9"
)

type Options struct {
	Stream       string        // default: "stream:wallet"
	Group        string        // default: "wallet_cg"
	Block        time.Duration // default: 5s
	Batch        int64         // default: 100
	MinIdle      time.Duration // default: 30s
	TrimAfterAck bool          // optional: XDEL after ack
}

type WalletStreamWorker struct {
	rdb  redis.UniversalClient
	repo *repository.WalletRepository
	opt  Options
}

func NewWalletStreamWorker(rdb redis.UniversalClient, repo *repository.WalletRepository, opt *Options) *WalletStreamWorker {
	o := Options{
		Stream:  "stream:wallet",
		Group:   "wallet_cg",
		Block:   5 * time.Second,
		Batch:   100,
		MinIdle: 30 * time.Second,
	}
	if opt != nil {
		if opt.Stream != "" {
			o.Stream = opt.Stream
		}
		if opt.Group != "" {
			o.Group = opt.Group
		}
		if opt.Block != 0 {
			o.Block = opt.Block
		}
		if opt.Batch != 0 {
			o.Batch = opt.Batch
		}
		if opt.MinIdle != 0 {
			o.MinIdle = opt.MinIdle
		}
		o.TrimAfterAck = opt.TrimAfterAck
	}
	return &WalletStreamWorker{rdb: rdb, repo: repo, opt: o}
}

func (w *WalletStreamWorker) ensureGroup(ctx context.Context) {
	err := w.rdb.XGroupCreateMkStream(ctx, w.opt.Stream, w.opt.Group, "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		logger.Errorf("❌ XGroupCreateMkStream error: %v", err)
	} else if err == nil {
		logger.Infof("✅ Created group %q on %s", w.opt.Group, w.opt.Stream)
	}
}

func (w *WalletStreamWorker) reclaimPending(ctx context.Context, consumer string) {
	start := "0-0"
	for {
		msgs, next, err := w.rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   w.opt.Stream,
			Group:    w.opt.Group,
			Consumer: consumer,
			MinIdle:  w.opt.MinIdle,
			Start:    start,
			Count:    w.opt.Batch,
		}).Result()
		if err != nil {
			if err != redis.Nil {
				logger.Errorf("❌ XAutoClaim error: %v", err)
			}
			return
		}
		if len(msgs) == 0 {
			return
		}
		for _, m := range msgs {
			w.handleMessage(ctx, consumer, m)
		}
		start = next
	}
}

func (w *WalletStreamWorker) handleMessage(ctx context.Context, consumer string, m redis.XMessage) {
	f := m.Values

	ev, _ := getStr(f, "type")
	if ev == "" {
		ev, _ = getStr(f, "ev")
	}
	if ev != "DEPOSIT" {
		_ = w.rdb.XAck(ctx, w.opt.Stream, w.opt.Group, m.ID).Err()
		if w.opt.TrimAfterAck {
			_ = w.rdb.XDel(ctx, w.opt.Stream, m.ID).Err()
		}
		return
	}

	userIDStr, _ := getStr(f, "user_id")
	currency, _ := getStr(f, "currency")
	txID, _ := getStr(f, "tx_id")
	amountStr, _ := getStr(f, "amount")
	metaStr, _ := getStr(f, "meta")

	uid, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		logger.Errorf("❌ worker parse user_id error (msg=%s): %v", m.ID, err)
		return
	}
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		logger.Errorf("❌ worker parse amount error (msg=%s): %v", m.ID, err)
		return
	}
	currency = strings.ToUpper(currency)

	network := "NATIVE"
	if metaStr != "" {
		var meta map[string]any
		if json.Unmarshal([]byte(metaStr), &meta) == nil {
			if v, ok := meta["network"].(string); ok && v != "" {
				network = strings.ToUpper(v)
			}
		}
	}

	if err := w.repo.UpsertDeposit(ctx, uid, currency, network, amount); err != nil {
		logger.Errorf("❌ UpsertDeposit error (tx=%s msg=%s): %v", txID, m.ID, err)
		return // no ack -> retry later
	}

	if err := w.rdb.XAck(ctx, w.opt.Stream, w.opt.Group, m.ID).Err(); err != nil {
		logger.Errorf("❌ XAck error (msg=%s): %v", m.ID, err)
	}

	if w.opt.TrimAfterAck {
		_ = w.rdb.XDel(ctx, w.opt.Stream, m.ID).Err()
	}
	logger.Infof("✅ Handled message %s for consumer %s", m.ID, consumer)
}

func getStr(m map[string]interface{}, key string) (string, bool) {
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case string:
			return t, true
		case []byte:
			return string(t), true
		}
	}
	return "", false
}

func (w *WalletStreamWorker) Run(ctx context.Context, consumerName string) {
	w.ensureGroup(ctx)
	w.reclaimPending(ctx, consumerName)

	backoff := 200 * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		res, err := w.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    w.opt.Group,
			Consumer: consumerName,
			Streams:  []string{w.opt.Stream, ">"},
			Count:    w.opt.Batch,
			Block:    w.opt.Block,
		}).Result()
		if err != nil {
			if err != redis.Nil {
				logger.Errorf("❌ XReadGroup error: %v", err)
				time.Sleep(backoff)
				if backoff < 5*time.Second {
					backoff *= 2
				}
			}
			continue
		}
		backoff = 200 * time.Millisecond
		for _, strm := range res {
			for _, msg := range strm.Messages {
				w.handleMessage(ctx, consumerName, msg)
			}
		}
	}
}

func ConsumerName(instance string, i int) string {
	if instance == "" {
		instance = "app"
	}
	return fmt.Sprintf("%s-%d", instance, i)
}
