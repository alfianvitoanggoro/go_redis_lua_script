package dto

// DepositInput: request yang masuk ke usecase.
// amount = minor units (mis. IDR: 1 rupiah -> 1, USD: 1 cent -> 1, BTC: 1 sat -> 1)
type DepositInput struct {
	UserID   int64          `json:"user_id" validate:"required"`
	Currency string         `json:"currency" validate:"required,uppercase"`
	TxID     string         `json:"tx_id" validate:"required"` // dipakai untuk FIFO/reqID & idemp log (nanti)
	Amount   int64          `json:"amount" validate:"required,gt=0"`
	Meta     map[string]any `json:"meta,omitempty"`
}

type DepositOutput struct {
	Code         int64  `json:"code"` // 1=applied, 0=idempotent, -2=invalid
	Applied      bool   `json:"applied"`
	RedisBalance int64  `json:"redis_balance"` // minor units di Redis
	Currency     string `json:"currency"`
}
