-- Add provider_transaction_id field to store provider's transaction UUID
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS provider_transaction_id VARCHAR(255);

-- Create index for fast webhook lookups
CREATE INDEX IF NOT EXISTS idx_transactions_provider_transaction_id
ON transactions(provider_transaction_id)
WHERE provider_transaction_id IS NOT NULL;
