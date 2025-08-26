package async

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	"grls/internal/infrastructure/repository"
	"grls/internal/store"
	"grls/pkg/logger"
)

type Processor struct {
	Rdb        redis.UniversalClient
	Repo       *repository.WalletRepository
	Queue      *store.RedisQueue
	BRPopBlock time.Duration
	DBExecTO   time.Duration
}

func NewProcessor(rdb redis.UniversalClient, repo *repository.WalletRepository, q *store.RedisQueue) *Processor {
	return &Processor{
		Rdb:        rdb,
		Repo:       repo,
		Queue:      q,
		BRPopBlock: 5 * time.Second,
		DBExecTO:   2 * time.Second,
	}
}

func (p *Processor) Run(ctx context.Context) {
	logger.Info("async processor started")
	defer logger.Info("async processor stopped")

	readyKey := p.Queue.ReadyKeyName()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Ambil queue user yang siap (res[1] = "q:{user}")
		res, err := p.Rdb.BRPop(ctx, p.BRPopBlock, readyKey).Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			logger.Warnf("BRPOP err: %v", err)
			continue
		}
		if len(res) != 2 {
			continue
		}
		qKey := res[1]
		user, ok := parseUserFromQueueKey(qKey)
		if !ok {
			logger.Warnf("cannot parse user from key=%s", qKey)
			continue
		}

		// Baca head payload (tanpa pop)
		head, err := p.Rdb.LIndex(ctx, qKey, 0).Result()
		if err == redis.Nil || head == "" {
			continue
		}
		if err != nil {
			logger.Warnf("LINDEX err user=%s: %v", user, err)
			continue
		}

		var payload store.DepositPayload
		if err := json.Unmarshal([]byte(head), &payload); err != nil {
			logger.Errorf("JSON decode err user=%s head=%q: %v", user, head, err)
			// buang item buruk agar tidak macet
			_, _ = p.Queue.ReleaseAndPromote(context.Background(), user)
			continue
		}

		// Commit ke DB (as-is integer → decimal)
		dbCtx, cancel := context.WithTimeout(context.Background(), p.DBExecTO)
		err = p.Repo.UpsertDepositDecimal(dbCtx, mustParseInt64(payload.UserID), strings.ToUpper(payload.Currency), decimal.NewFromInt(payload.Amount))
		cancel()
		if err != nil {
			logger.Errorf("DB err user=%s cur=%s amt=%d tx=%s: %v", payload.UserID, payload.Currency, payload.Amount, payload.TxID, err)
			// retry: dorong lagi qKey ke ready agar diambil ulang setelah jeda
			_ = p.Rdb.LPush(context.Background(), readyKey, qKey).Err()
			time.Sleep(20 * time.Millisecond)
			continue
		}

		// Sukses → release & promote
		if _, err := p.Queue.ReleaseAndPromote(context.Background(), user); err != nil {
			logger.Warnf("release warn user=%s: %v", user, err)
		}
	}
}

func parseUserFromQueueKey(qKey string) (string, bool) {
	// ekspektasi: "q:{<user>}"
	i := strings.Index(qKey, "{")
	j := strings.LastIndex(qKey, "}")
	if i == -1 || j == -1 || j <= i+1 {
		return "", false
	}
	return qKey[i+1 : j], true
}

func mustParseInt64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}
