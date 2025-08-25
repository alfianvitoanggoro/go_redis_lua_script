-- 000001_create_wallets_table.down.sql
DROP TRIGGER IF EXISTS trg_wallets_currency_upper ON wallets;
DROP FUNCTION IF EXISTS normalize_currency();

DROP TRIGGER IF EXISTS trg_wallets_updated_at ON wallets;
DROP FUNCTION IF EXISTS set_updated_at();

DROP INDEX IF EXISTS idx_wallets_user_id;
DROP TABLE IF EXISTS wallets;
