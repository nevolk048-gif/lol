-- Add MajorPay provider with sandbox configuration
INSERT INTO providers (id, name, api_key, secret_key, merchant_id, base_url, webhook_url, status, is_sandbox) VALUES
('b0000000-0000-0000-0000-000000000005', 'MajorPay', 'pk_9ec270e742bf8', 'sk_majorpay_secret_key_here', NULL, 'https://api.majorpay.io/api', 'https://api.majorpay.io/api/webhook', 'ACTIVE', true)
ON CONFLICT (id) DO UPDATE SET
    base_url = EXCLUDED.base_url,
    webhook_url = EXCLUDED.webhook_url,
    api_key = EXCLUDED.api_key,
    status = EXCLUDED.status,
    updated_at = NOW();

-- Add MajorPay requisites for RUB/RU
INSERT INTO requisites (id, provider_id, bank_name, holder_name, account_number, currency, country, daily_limit, used_limit, status, is_sandbox) VALUES
('d0000000-0000-0000-0000-000000000007', 'b0000000-0000-0000-0000-000000000005', 'Sberbank', 'MajorPay LLC', '****1234', 'RUB', 'RU', 5000000, 0, 'ACTIVE', true),
('d0000000-0000-0000-0000-000000000008', 'b0000000-0000-0000-0000-000000000005', 'Tinkoff', 'MajorPay LLC', '****5678', 'RUB', 'RU', 3000000, 0, 'ACTIVE', true)
ON CONFLICT (id) DO UPDATE SET
    daily_limit = EXCLUDED.daily_limit,
    used_limit = 0,
    status = EXCLUDED.status,
    updated_at = NOW();

-- Add routing rule for MajorPay (RUB/RU with priority)
INSERT INTO route_rules (id, priority, weight, country, currency, provider_id, status, is_fallback, is_sandbox) VALUES
('e0000000-0000-0000-0000-000000000007', 1, 100, 'RU', 'RUB', 'b0000000-0000-0000-0000-000000000005', 'ACTIVE', false, true)
ON CONFLICT (id) DO UPDATE SET
    weight = EXCLUDED.weight,
    status = EXCLUDED.status,
    updated_at = NOW();

COMMENT ON TABLE providers IS 'Payment providers including MajorPay for RUB processing';
