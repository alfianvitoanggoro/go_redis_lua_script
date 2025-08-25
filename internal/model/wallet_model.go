package model

import "time"

// Hindari float untuk balance. Simpan sebagai string agar akurat saat scan dari NUMERIC.
type Wallet struct {
	ID        int64     `json:"id"         gorm:"column:id;primaryKey"`
	UserID    int64     `json:"user_id"    gorm:"column:user_id;not null"`
	Currency  string    `json:"currency"   gorm:"column:currency;type:VARCHAR(10);not null"`
	Balance   string    `json:"balance"    gorm:"column:balance;type:NUMERIC(20,8);not null;default:0"`
	IsActive  bool      `json:"is_active"  gorm:"column:is_active;not null;default:true"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;type:timestamptz;not null;default:now()"`
}

func (Wallet) TableName() string { return "wallets" }
