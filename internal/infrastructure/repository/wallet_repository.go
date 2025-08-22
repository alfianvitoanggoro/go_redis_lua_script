package repository

import (
	"context"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"grls/pkg/logger"
)

type WalletRepository struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
}

func NewWalletRepository(dbWrite *gorm.DB, dbRead *gorm.DB) *WalletRepository {
	return &WalletRepository{dbWrite: dbWrite, dbRead: dbRead}
}

/* --- helper konversi minor→NUMERIC(20,8) --- */
var currencyScale = map[string]int{
	"IDR": 0, "USD": 2, "SGD": 2, "EUR": 2,
	"USDT": 6, "BTC": 8, "ETH": 8, // clamp 8 agar muat
}

func minorToNumericStr(amount int64, scale int) string {
	if scale <= 0 {
		return strconv.FormatInt(amount, 10)
	}
	neg := amount < 0
	if neg {
		amount = -amount
	}
	s := strconv.FormatInt(amount, 10)
	if len(s) <= scale {
		s = strings.Repeat("0", scale-len(s)+1) + s
	}
	intPart := s[:len(s)-scale]
	fracPart := s[len(s)-scale:]
	if neg {
		return "-" + intPart + "." + fracPart
	}
	return intPart + "." + fracPart
}

/* --- UpsertDeposit: tambah balance di DB --- */
func (r *WalletRepository) UpsertDeposit(ctx context.Context, userID int64, currency, network string, amountMinor int64) error {
	cur := strings.ToUpper(currency)
	if network == "" {
		network = "NATIVE"
	}
	scale := currencyScale[cur]
	if scale > 8 {
		scale = 8
	}
	amountStr := minorToNumericStr(amountMinor, scale)

	sql := `
		INSERT INTO wallets (user_id, currency, network, balance, is_active)
		VALUES (?, ?, ?, ?, TRUE)
		ON CONFLICT (user_id, currency, network)
		DO UPDATE SET balance = wallets.balance + EXCLUDED.balance, updated_at = NOW()
	`
	if err := r.dbWrite.WithContext(ctx).Exec(sql, userID, cur, network, amountStr).Error; err != nil {
		logger.Errorf("WalletRepository.UpsertDeposit: %v", err)
		return err
	}
	return nil
}
