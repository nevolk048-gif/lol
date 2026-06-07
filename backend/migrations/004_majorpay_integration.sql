-- Add MajorPay integration support
-- merchant_customer_id for Payer Affinity
-- Store amounts in minor units (kopecks) but keep decimal for compatibility

ALTER TABLE transactions ADD COLUMN IF NOT EXISTS merchant_customer_id VARCHAR(255);
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS payment_method VARCHAR(50);
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);

-- Index for Payer Affinity lookups
CREATE INDEX IF NOT EXISTS idx_transactions_merchant_customer_id ON transactions(merchant_customer_id) WHERE merchant_customer_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_transactions_idempotency_key ON transactions(idempotency_key) WHERE idempotency_key IS NOT NULL;

-- Add fields for requisite history tracking (for Payer Affinity)
CREATE TABLE IF NOT EXISTS customer_requisite_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    merchant_customer_id VARCHAR(255) NOT NULL,
    requisite_id UUID NOT NULL REFERENCES requisites(id),
    casino_id UUID NOT NULL REFERENCES casinos(id),
    last_success_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    success_count INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(merchant_customer_id, requisite_id, casino_id)
);

CREATE INDEX IF NOT EXISTS idx_customer_requisite_history_lookup
ON customer_requisite_history(merchant_customer_id, casino_id, last_success_at DESC);

-- Comments
COMMENT ON COLUMN transactions.merchant_customer_id IS 'Customer ID for Payer Affinity algorithm';
COMMENT ON COLUMN transactions.payment_method IS 'Payment method hint (auto, card, sbp, etc)';
COMMENT ON COLUMN transactions.idempotency_key IS 'Idempotency key for duplicate prevention';
COMMENT ON TABLE customer_requisite_history IS 'Tracks successful customer-requisite pairs for Payer Affinity';
