package usecase

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"grls/internal/infrastructure/repository"
	"grls/internal/modules/wallet/dto"
	"grls/internal/modules/wallet/store"
)

type WalletUsecase struct {
	repo *repository.WalletRepository
	rds  *store.RedisWalletStore
}

func NewWalletUsecase(repo *repository.WalletRepository, rds *store.RedisWalletStore) *WalletUsecase {
	return &WalletUsecase{repo: repo, rds: rds}
}

func (u *WalletUsecase) Deposit(ctx context.Context, in dto.DepositInput) (dto.DepositOutput, error) {
	if in.UserID == "" || in.Currency == "" || in.TxID == "" || in.Amount <= 0 {
		return dto.DepositOutput{Code: -2}, errors.New("invalid input")
	}
	cur := strings.ToUpper(in.Currency)
	net := in.Network
	if net == "" {
		net = "NATIVE"
	}

	// 1) Redis Lua (atomic + idempotent)
	res, err := u.rds.Deposit(ctx, in.UserID, cur, in.TxID, in.Amount, in.Meta)
	if err != nil {
		return dto.DepositOutput{}, err
	}

	// 2) Jika applied, upsert ke DB
	if res.Applied {
		uid, err := strconv.ParseInt(in.UserID, 10, 64)
		if err != nil {
			return dto.DepositOutput{Code: res.Code, Applied: res.Applied, RedisBalance: res.Balance, Currency: cur, Network: net},
				errors.New("user_id must be numeric")
		}
		if err := u.repo.UpsertDeposit(ctx, uid, cur, net, in.Amount); err != nil {
			return dto.DepositOutput{Code: res.Code, Applied: res.Applied, RedisBalance: res.Balance, Currency: cur, Network: net}, err
		}
	}

	return dto.DepositOutput{
		Code:         res.Code,
		Applied:      res.Applied,
		RedisBalance: res.Balance,
		Currency:     cur,
		Network:      net,
	}, nil
}
