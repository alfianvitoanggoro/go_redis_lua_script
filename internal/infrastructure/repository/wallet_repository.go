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

// UpsertDepositDecimal: tambah saldo dengan nilai desimal "as-is" (dibatasi 8 dp).
func (r *WalletRepository) UpsertDepositDecimal(
	ctx context.Context,
	userID int64,
	currency string,
	amount decimal.Decimal,
) error {
	cur := strings.ToUpper(currency)
	amount = amount.Round(8) // match kolom NUMERIC(20,8)

	const sql = `
		INSERT INTO wallets (user_id, currency, balance, is_active)
		VALUES (?, ?, ?, TRUE)
		ON CONFLICT (user_id, currency)
		DO UPDATE SET
			balance    = wallets.balance + EXCLUDED.balance,
			updated_at = NOW()
	`
	return r.dbWrite.WithContext(ctx).Exec(sql, userID, cur, amount.String()).Error
}

// UpsertDepositInt: versi uji coba, amount integer "as-is" (tanpa desimal).
func (r *WalletRepository) UpsertDepositInt(
	ctx context.Context,
	userID int64,
	currency string,
	amount int64,
) error {
	return r.UpsertDepositDecimal(ctx, userID, currency, decimal.NewFromInt(amount))
}

// GetWallet: ambil dompet user+currency (berguna untuk verifikasi test).
func (r *WalletRepository) GetWallet(ctx context.Context, userID int64, currency string) (struct {
	ID        int64
	UserID    int64
	Currency  string
	Balance   string
	IsActive  bool
	CreatedAt string
	UpdatedAt string
}, error) {
	var out struct {
		ID        int64
		UserID    int64
		Currency  string
		Balance   string
		IsActive  bool
		CreatedAt string
		UpdatedAt string
	}
	err := r.dbRead.WithContext(ctx).
		Raw(`
			SELECT
				id,
				user_id,
				currency,
				balance::text AS balance,
				is_active,
				created_at::text,
				updated_at::text
			FROM wallets
			WHERE user_id = ? AND currency = ?
			LIMIT 1
		`, userID, strings.ToUpper(currency)).
		Scan(&out).Error
	return out, err
}
