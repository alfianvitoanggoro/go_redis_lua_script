package grpcserver

import (
	"context"
	"strings"
	"time"

	"google.golang.org/grpc"

	"grls/internal/store"
	"grls/pkg/logger"
	walletv1 "grls/pkg/proto/wallet/v1"
)

const enqueueTO = 1500 * time.Millisecond

type server struct {
	walletv1.UnimplementedWalletServiceServer
	queue *store.RedisQueue
}

func NewWalletServiceServer(queue *store.RedisQueue) *server {
	return &server{queue: queue}
}

func RegisterWalletService(s *grpc.Server, queue *store.RedisQueue) {
	walletv1.RegisterWalletServiceServer(s, NewWalletServiceServer(queue))
}

var _ walletv1.WalletServiceServer = (*server)(nil)

func (s *server) Deposit(ctx context.Context, req *walletv1.DepositRequest) (*walletv1.DepositResponse, error) {
	if req.GetUserId() == "" || req.GetCurrency() == "" || req.GetTxId() == "" {
		return &walletv1.DepositResponse{
			Status:  walletv1.DepositResponse_FAILED,
			Message: "invalid request: user_id/currency/tx_id required",
		}, nil
	}
	if req.GetAmount() <= 0 {
		return &walletv1.DepositResponse{
			Status:  walletv1.DepositResponse_FAILED,
			Message: "invalid amount: must be > 0",
		}, nil
	}

	payload := store.DepositPayload{
		UserID:   req.GetUserId(),
		Currency: strings.ToUpper(req.GetCurrency()),
		Amount:   req.GetAmount(),
		TxID:     req.GetTxId(),
	}

	enqCtx, cancel := context.WithTimeout(context.Background(), enqueueTO)
	_, err := s.queue.EnqueueDeposit(enqCtx, payload)
	cancel()
	if err != nil {
		logger.Errorf("enqueue error user=%s cur=%s amt=%d tx=%s: %v",
			payload.UserID, payload.Currency, payload.Amount, payload.TxID, err)
		return &walletv1.DepositResponse{
			Status:  walletv1.DepositResponse_FAILED,
			Message: "queue error",
		}, nil
	}

	return &walletv1.DepositResponse{
		Status:  walletv1.DepositResponse_SUCCESS,
		Message: "accepted",
	}, nil
}
