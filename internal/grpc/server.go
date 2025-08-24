package grpc

import (
	"context"
	"strings"

	"google.golang.org/grpc"

	"grls/pkg/logger"
	walletv1 "grls/pkg/proto/wallet/v1"

	wstore "grls/internal/modules/wallet/store"
)

type server struct {
	walletv1.UnimplementedWalletServiceServer
	rds *wstore.RedisWalletStore
}

// NewWalletServiceServer creates a new gRPC server instance with Redis + Repository deps.
func NewWalletServiceServer(rds *wstore.RedisWalletStore) *server {
	return &server{rds: rds}
}

// RegisterWalletService registers WalletService into the given gRPC server.
func RegisterWalletService(s *grpc.Server, rds *wstore.RedisWalletStore) {
	walletv1.RegisterWalletServiceServer(s, NewWalletServiceServer(rds))
}

// package grpc

func (s *server) Deposit(ctx context.Context, req *walletv1.DepositRequest) (*walletv1.DepositResponse, error) {
	if req.GetUserId() == "" || req.GetCurrency() == "" || req.GetTxId() == "" || req.GetAmount() <= 0 {
		logger.Errorf("❌ invalid deposit request: %+v", req)
		return &walletv1.DepositResponse{Code: -2, Applied: false}, nil
	}

	cur := strings.ToUpper(req.GetCurrency())
	netw := req.GetNetwork()
	if netw == "" {
		netw = "NATIVE"
	}

	// sisipkan network ke meta agar worker bisa baca
	meta := req.GetMeta()
	if meta == nil {
		meta = map[string]string{}
	}
	meta["network"] = netw

	// Only hit Redis (atomic + idempotent)
	res, err := s.rds.Deposit(ctx, req.GetUserId(), cur, req.GetTxId(), req.GetAmount(), mapStringToAny(meta))
	if err != nil {
		logger.Errorf("❌ failed deposit %+v: %v", req, err)
		return &walletv1.DepositResponse{Code: 0, Applied: false}, nil
	}

	logger.Infof("✅ successful deposit %+v: %+v", req, res)
	logger.WriteLogToFile("success", "WalletServer.Deposit", map[string]any{"req": req, "code": res.Code, "applied": res.Applied}, nil)
	return &walletv1.DepositResponse{Code: res.Code, Applied: res.Applied}, nil
}

// helper: map[string]string → map[string]any
func mapStringToAny(in map[string]string) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
