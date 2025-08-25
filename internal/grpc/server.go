// internal/grpc/server.go
package grpcserver

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc"

	"grls/internal/infrastructure/repository"
	"grls/internal/store"
	"grls/pkg/logger"
	walletv1 "grls/pkg/proto/wallet/v1"
)

type server struct {
	walletv1.UnimplementedWalletServiceServer
	repo         *repository.WalletRepository
	fifo         *store.FIFOLock
	waitMax      time.Duration
	pollInterval time.Duration
}

// NewWalletServiceServer membuat instance gRPC server dengan dependensi Repo + FIFO.
func NewWalletServiceServer(repo *repository.WalletRepository, fifo *store.FIFOLock) *server {
	return &server{
		repo:         repo,
		fifo:         fifo,
		waitMax:      5 * time.Second,       // max tunggu giliran
		pollInterval: 10 * time.Millisecond, // interval polling promote
	}
}

// RegisterWalletService mendaftarkan WalletService ke gRPC server.
func RegisterWalletService(s *grpc.Server, repo *repository.WalletRepository, fifo *store.FIFOLock) {
	walletv1.RegisterWalletServiceServer(s, NewWalletServiceServer(repo, fifo))
}

// Deposit flow:
// 1) Enqueue & coba acquire per-user FIFO (Redis+Lua)
// 2) Saat dapat giliran, upsert saldo ke DB (balance += amount)
// 3) Release & promote next (selalu, agar antrian tidak macet)
//
// NOTE: amount = integer "as-is" (uji coba), TIDAK pakai network.
func (s *server) Deposit(ctx context.Context, req *walletv1.DepositRequest) (*walletv1.DepositResponse, error) {
	// ---- Validasi dasar ----
	if req.GetUserId() == "" || req.GetCurrency() == "" || req.GetTxId() == "" {
		return &walletv1.DepositResponse{
			Status:  walletv1.DepositResponse_FAILED,
			Message: "invalid request: user_id/currency/tx_id required",
		}, nil
	}
	amt := req.GetAmount()
	if amt <= 0 {
		return &walletv1.DepositResponse{
			Status:  walletv1.DepositResponse_FAILED,
			Message: "invalid amount: must be > 0",
		}, nil
	}

	cur := strings.ToUpper(req.GetCurrency())
	userStr := req.GetUserId()
	txID := req.GetTxId()

	// Parse user_id (untuk DB)
	userID, err := strconv.ParseInt(userStr, 10, 64)
	if err != nil {
		return &walletv1.DepositResponse{
			Status:  walletv1.DepositResponse_FAILED,
			Message: "invalid user_id: not int64",
		}, nil
	}

	// ---- 1) Enqueue + coba acquire giliran (atomic via Lua) ----
	ctxLua, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	acq, err := s.fifo.EnqueueAndTryAcquire(ctxLua, userStr, txID)
	cancel()
	if err != nil {
		logger.Errorf("Redis enqueue/acquire error: %v", err)
		return &walletv1.DepositResponse{
			Status:  walletv1.DepositResponse_FAILED,
			Message: "queue error",
		}, nil
	}
	// belum dapat? tunggu FIFO (poll ringan)
	if !acq {
		ok, err := s.fifo.WaitUntilAcquired(ctx, userStr, txID, s.waitMax, s.pollInterval)
		if err != nil {
			logger.Errorf("Redis promote error: %v", err)
			return &walletv1.DepositResponse{
				Status:  walletv1.DepositResponse_FAILED,
				Message: "queue error",
			}, nil
		}
		if !ok {
			// timeout menunggu giliran → minta klien retry
			logger.Warnf("FIFO wait timeout user=%s tx=%s", userStr, txID)
			return &walletv1.DepositResponse{
				Status:  walletv1.DepositResponse_FAILED,
				Message: "queue busy, retry later",
			}, nil
		}
	}

	// Pastikan selalu release & promote agar queue lanjut,
	// apapun hasil DB (sukses / gagal).
	defer func() {
		_ = s.fifo.ReleaseAndPromote(context.Background(), userStr, txID)
	}()

	// ---- 2) Apply ke DB (amount integer as-is → decimal) ----
	decAmt := decimal.NewFromInt(amt)
	if err := s.repo.UpsertDepositDecimal(ctx, userID, cur, decAmt); err != nil {
		logger.Errorf("DB UpsertDepositDecimal error user=%d cur=%s amt=%s: %v",
			userID, cur, decAmt.String(), err)
		return &walletv1.DepositResponse{
			Status:  walletv1.DepositResponse_FAILED,
			Message: "db error",
		}, nil
	}

	logger.Infof("✅ Deposit applied user=%d cur=%s amt=%s tx=%s",
		userID, cur, decAmt.String(), txID)

	return &walletv1.DepositResponse{
		Status:  walletv1.DepositResponse_SUCCESS,
		Message: "deposit applied",
	}, nil
}
