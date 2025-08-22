package dto

type DepositInput struct {
	UserID   string         `json:"user_id"   validate:"required,numeric"`
	Currency string         `json:"currency"  validate:"required,uppercase"`
	Network  string         `json:"network"   validate:"omitempty,uppercase"`
	TxID     string         `json:"tx_id"     validate:"required"`
	Amount   int64          `json:"amount"    validate:"required,gt=0"` // minor units (integer)
	Meta     map[string]any `json:"meta,omitempty"`
}

type DepositOutput struct {
	Code         int64  `json:"code"` // 1=applied, 0=idempotent, -2=invalid
	Applied      bool   `json:"applied"`
	RedisBalance int64  `json:"redis_balance"` // minor units di Redis
	Currency     string `json:"currency"`
	Network      string `json:"network"`
}
