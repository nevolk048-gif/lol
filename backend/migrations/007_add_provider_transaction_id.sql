-- Add provider_transaction_id field to store MajorPay UUID
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS provider_transaction_id VARCHAR(255);

-- Create index for webhook lookups
CREATE INDEX IF NOT EXISTS idx_transactions_provider_transaction_id
ON transactions(provider_transaction_id)
WHERE provider_transaction_id IS NOT NULL;

COMMENT ON COLUMN transactions.provider_transaction_id IS 'Transaction ID from provider (e.g., MajorPay UUID pay_4d0b1f...)';
