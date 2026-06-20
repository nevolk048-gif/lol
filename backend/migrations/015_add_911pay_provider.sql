-- Add 911pay provider with production credentials
-- Merchant ID: 8f323aa6-4aff-4f71-9611-767ffc5e1d4a
-- API: https://911pay.cc/api/merchant/order
--
-- ВАЖНО: после добавления задайте webhook_url равным
--   https://<ваш-сервер>/api/v1/webhook/pay911
-- чтобы 911pay присылал колбэки о смене статуса ордеров.

INSERT INTO providers (
    id,
    name,
    api_key,
    secret_key,
    merchant_id,
    base_url,
    webhook_url,
    status,
    is_sandbox,
    traffic_enabled
)
VALUES (
    'c1000000-0000-0000-0000-000000000911',
    '911pay',
    -- api_key хранит merchant_id (используется как Access-Token в заголовке)
    '8f323aa6-4aff-4f71-9611-767ffc5e1d4a',
    -- secret_key — для верификации HMAC-подписи webhook-колбэков
    'KrZTUxpQC9WkyFw75bh1CNegoYf7NlomPnaGeiew',
    -- merchant_id — UUID мерчанта на стороне 911pay
    '8f323aa6-4aff-4f71-9611-767ffc5e1d4a',
    -- base_url — адрес API 911pay (оставить пустым для использования дефолтного https://911pay.cc)
    '',
    -- webhook_url — наш callback URL, который мы передаём 911pay при создании ордера.
    -- Заполните после деплоя: https://<ваш-сервер>/api/v1/webhook/pay911
    '',
    'ACTIVE',
    false,  -- не sandbox (реальные кредентиалы)
    true    -- traffic_enabled
)
ON CONFLICT (id) DO UPDATE SET
    name        = EXCLUDED.name,
    api_key     = EXCLUDED.api_key,
    secret_key  = EXCLUDED.secret_key,
    merchant_id = EXCLUDED.merchant_id,
    base_url    = EXCLUDED.base_url,
    status      = EXCLUDED.status,
    updated_at  = NOW();

-- Роутинговое правило для 911pay: RUB / RU, приоритет 2 (MajorPay = 1)
INSERT INTO route_rules (
    id,
    priority,
    weight,
    country,
    currency,
    provider_id,
    status,
    is_fallback,
    is_sandbox
)
VALUES (
    'f1000000-0000-0000-0000-000000000911',
    2,
    100,
    'RU',
    'RUB',
    'c1000000-0000-0000-0000-000000000911',
    'ACTIVE',
    false,
    false  -- production rule
)
ON CONFLICT (id) DO UPDATE SET
    weight     = EXCLUDED.weight,
    status     = EXCLUDED.status,
    updated_at = NOW();
