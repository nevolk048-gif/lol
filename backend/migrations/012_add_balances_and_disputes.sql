-- Migration: Add balances for providers and merchants, and disputes system
-- Created: 2026-06-08

-- Provider Balances
CREATE TABLE IF NOT EXISTS provider_balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    balance DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    frozen_balance DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(provider_id, currency)
);

CREATE INDEX idx_provider_balances_provider ON provider_balances(provider_id);

-- Provider Balance Transactions
CREATE TABLE IF NOT EXISTS provider_balance_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_balance_id UUID NOT NULL REFERENCES provider_balances(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- DEPOSIT, WITHDRAWAL, FEE, ADJUSTMENT, FREEZE, UNFREEZE, COMMISSION
    amount DECIMAL(15, 2) NOT NULL,
    balance_before DECIMAL(15, 2) NOT NULL,
    balance_after DECIMAL(15, 2) NOT NULL,
    description TEXT,
    reference_type VARCHAR(50), -- TRANSACTION, DISPUTE, PAYOUT, MANUAL
    reference_id UUID,
    performed_by UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_provider_balance_txs_provider ON provider_balance_transactions(provider_id);
CREATE INDEX idx_provider_balance_txs_created ON provider_balance_transactions(created_at DESC);
CREATE INDEX idx_provider_balance_txs_reference ON provider_balance_transactions(reference_type, reference_id);

-- Merchant (Casino) Balances
CREATE TABLE IF NOT EXISTS merchant_balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    casino_id UUID NOT NULL REFERENCES casinos(id) ON DELETE CASCADE,
    balance DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    frozen_balance DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(casino_id, currency)
);

CREATE INDEX idx_merchant_balances_casino ON merchant_balances(casino_id);

-- Merchant Balance Transactions
CREATE TABLE IF NOT EXISTS merchant_balance_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_balance_id UUID NOT NULL REFERENCES merchant_balances(id) ON DELETE CASCADE,
    casino_id UUID NOT NULL REFERENCES casinos(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- DEPOSIT, WITHDRAWAL, FEE, PAYOUT, REFUND, CHARGEBACK
    amount DECIMAL(15, 2) NOT NULL,
    balance_before DECIMAL(15, 2) NOT NULL,
    balance_after DECIMAL(15, 2) NOT NULL,
    description TEXT,
    reference_type VARCHAR(50), -- TRANSACTION, PAYOUT, DISPUTE
    reference_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_merchant_balance_txs_casino ON merchant_balance_transactions(casino_id);
CREATE INDEX idx_merchant_balance_txs_created ON merchant_balance_transactions(created_at DESC);
CREATE INDEX idx_merchant_balance_txs_reference ON merchant_balance_transactions(reference_type, reference_id);

-- Disputes
CREATE TABLE IF NOT EXISTS disputes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    casino_id UUID NOT NULL REFERENCES casinos(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'NEW', -- NEW, UNDER_REVIEW, AWAITING_PROVIDER_RESPONSE, MERCHANT_WON, PROVIDER_WON, CLOSED
    reason TEXT NOT NULL,
    amount DECIMAL(15, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    created_by UUID,
    resolved_by UUID,
    resolved_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_disputes_transaction ON disputes(transaction_id);
CREATE INDEX idx_disputes_provider ON disputes(provider_id);
CREATE INDEX idx_disputes_casino ON disputes(casino_id);
CREATE INDEX idx_disputes_status ON disputes(status);
CREATE INDEX idx_disputes_created ON disputes(created_at DESC);

-- Dispute Messages
CREATE TABLE IF NOT EXISTS dispute_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dispute_id UUID NOT NULL REFERENCES disputes(id) ON DELETE CASCADE,
    sender_type VARCHAR(20) NOT NULL, -- ADMIN, PROVIDER, MERCHANT
    sender_id UUID NOT NULL,
    message TEXT NOT NULL,
    attachments JSONB, -- Array of file URLs/metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dispute_messages_dispute ON dispute_messages(dispute_id);
CREATE INDEX idx_dispute_messages_created ON dispute_messages(created_at);

-- Dispute History (audit log)
CREATE TABLE IF NOT EXISTS dispute_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dispute_id UUID NOT NULL REFERENCES disputes(id) ON DELETE CASCADE,
    action VARCHAR(100) NOT NULL, -- STATUS_CHANGED, MESSAGE_ADDED, FILE_ATTACHED, etc.
    performed_by UUID,
    details JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dispute_history_dispute ON dispute_history(dispute_id);
CREATE INDEX idx_dispute_history_created ON dispute_history(created_at DESC);

-- Add traffic control fields to providers
ALTER TABLE providers ADD COLUMN IF NOT EXISTS traffic_enabled BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE providers ADD COLUMN IF NOT EXISTS traffic_disabled_reason TEXT;
ALTER TABLE providers ADD COLUMN IF NOT EXISTS traffic_disabled_at TIMESTAMP;
ALTER TABLE providers ADD COLUMN IF NOT EXISTS traffic_disabled_by UUID;

CREATE INDEX idx_providers_traffic_enabled ON providers(traffic_enabled);

-- Provider Traffic History
CREATE TABLE IF NOT EXISTS provider_traffic_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    action VARCHAR(20) NOT NULL, -- ENABLED, DISABLED
    reason TEXT,
    performed_by UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_provider_traffic_history_provider ON provider_traffic_history(provider_id);
CREATE INDEX idx_provider_traffic_history_created ON provider_traffic_history(created_at DESC);
