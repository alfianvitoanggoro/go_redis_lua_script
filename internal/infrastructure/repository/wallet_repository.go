package repository

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type WalletRepository struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
}

func NewWalletRepository(dbWrite *gorm.DB, dbRead *gorm.DB) *WalletRepository {
	return &WalletRepository{dbWrite: dbWrite, dbRead: dbRead}
}

// UpsertDepositDecimal: balance = balance + amount (NUMERIC(20,8)), idempotensi bisa ditambah nanti via ledger
func (r *WalletRepository) UpsertDepositDecimal(ctx context.Context, userID int64, currency string, amount decimal.Decimal) error {
	cur := strings.ToUpper(currency)
	amtStr := amount.String() // "as-is" (uji coba)

	sql := `
		INSERT INTO wallets (user_id, currency, balance, is_active)
		VALUES (?, ?, ?, TRUE)
		ON CONFLICT (user_id, currency)
		DO UPDATE SET
			balance    = wallets.balance + EXCLUDED.balance,
			updated_at = NOW()
	`
	return r.dbWrite.WithContext(ctx).Exec(sql, userID, cur, amtStr).Error
}
