CREATE TABLE IF NOT EXISTS wallets (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT        NOT NULL,
    currency    VARCHAR(10)   NOT NULL,
    network     VARCHAR(16)   NOT NULL DEFAULT 'NATIVE',
    balance     NUMERIC(20,8) NOT NULL DEFAULT 0,
    is_active   BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_wallet_identity UNIQUE (user_id, currency, network)
);
