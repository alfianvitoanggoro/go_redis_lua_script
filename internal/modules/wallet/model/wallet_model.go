package model

import "time"

// Wallet merepresentasikan row pada tabel wallets.
type Wallet struct {
	ID        int64     `json:"id" gorm:"column:id;primaryKey"`
	UserID    int64     `json:"user_id" gorm:"column:user_id;not null"`
	Currency  string    `json:"currency"  gorm:"column:currency;type:VARCHAR(10);primaryKey;not null"`
	Network   string    `json:"network"   gorm:"column:network;type:VARCHAR(16);primaryKey;not null;default:NATIVE"`
	Balance   string    `json:"balance"   gorm:"column:balance;type:numeric(20,8);not null;default:0"`
	IsActive  bool      `json:"is_active" gorm:"column:is_active;not null;default:true"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;type:timestamptz;not null;default:now()"`
}
