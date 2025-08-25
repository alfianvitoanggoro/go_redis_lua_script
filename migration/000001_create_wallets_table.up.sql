-- Tabel utama
CREATE TABLE IF NOT EXISTS wallets (
    id          BIGSERIAL     PRIMARY KEY,
    user_id     BIGINT        NOT NULL,
    currency    VARCHAR(10)   NOT NULL,
    balance     NUMERIC(20,8) NOT NULL DEFAULT 0,
    is_active   BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_wallet_identity  UNIQUE (user_id, currency),
    CONSTRAINT ck_balance_nonneg   CHECK (balance >= 0),
    CONSTRAINT ck_currency_format  CHECK (currency = UPPER(currency) AND currency ~ '^[A-Z0-9]{2,10}$')
);

-- Auto-update updated_at setiap UPDATE
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at := NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_wallets_updated_at ON wallets;
CREATE TRIGGER trg_wallets_updated_at
BEFORE UPDATE ON wallets
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- (Opsional) normalisasi currency ke UPPER agar dev tidak lupa
CREATE OR REPLACE FUNCTION normalize_currency() RETURNS TRIGGER AS $$
BEGIN
  NEW.currency := UPPER(NEW.currency);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_wallets_currency_upper ON wallets;
CREATE TRIGGER trg_wallets_currency_upper
BEFORE INSERT OR UPDATE ON wallets
FOR EACH ROW EXECUTE FUNCTION normalize_currency();

-- Index bantu query by user
CREATE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id);
