-- Update Demo Casino with integration fields
-- Run this after deploying merchant-demo to Railway

UPDATE casinos
SET
    merchant_id = 'casino_demo_001',
    base_url = 'https://merchant-demo.up.railway.app',  -- замени на реальный URL после деплоя
    secret_key = 'sk_demo_casino_secret_key_12345678',
    webhook_url = 'https://merchant-demo.up.railway.app/api/webhooks/paymentsgate'  -- замени на реальный URL
WHERE name = 'Demo Casino';

-- Verify the update
SELECT
    id,
    name,
    api_key,
    merchant_id,
    base_url,
    secret_key,
    webhook_url,
    status
FROM casinos
WHERE name = 'Demo Casino';
