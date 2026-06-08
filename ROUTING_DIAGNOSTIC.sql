-- Диагностика: Почему провайдер не участвует в маршрутизации
-- Выполните эти запросы для проверки

-- 1. Проверить статус провайдера (должен быть ACTIVE)
SELECT id, name, status, traffic_enabled, traffic_disabled_reason
FROM providers
WHERE id = 'YOUR_PROVIDER_ID';

-- 2. Проверить активные споры (НЕ должно быть активных споров)
SELECT COUNT(*) as active_disputes
FROM disputes
WHERE provider_id = 'YOUR_PROVIDER_ID'
  AND status IN ('NEW', 'UNDER_REVIEW', 'AWAITING_PROVIDER_RESPONSE');

-- 3. Проверить правила маршрутизации
SELECT rr.id, rr.priority, rr.weight, rr.status, rr.country, rr.currency
FROM route_rules rr
WHERE rr.provider_id = 'YOUR_PROVIDER_ID'
  AND rr.status = 'ACTIVE'
  AND rr.is_sandbox = false;  -- или true для песочницы

-- 4. Проверить реквизиты провайдера
SELECT id, bank_name, account_number, status, daily_limit, used_limit, currency, country
FROM requisites
WHERE provider_id = 'YOUR_PROVIDER_ID'
  AND status = 'ACTIVE';

-- 5. Полная проверка для конкретной транзакции
-- Заменить параметры на реальные значения
WITH check_params AS (
  SELECT
    'YOUR_PROVIDER_ID'::uuid as provider_id,
    'RUB' as currency,
    'RU' as country,
    false as is_sandbox
)
SELECT
  p.id,
  p.name,
  p.status as provider_status,
  p.traffic_enabled,
  p.traffic_disabled_reason,
  COUNT(DISTINCT rr.id) as active_rules,
  COUNT(DISTINCT r.id) as active_requisites,
  COUNT(DISTINCT d.id) as active_disputes
FROM providers p
CROSS JOIN check_params cp
LEFT JOIN route_rules rr ON rr.provider_id = p.id
  AND rr.status = 'ACTIVE'
  AND rr.is_sandbox = cp.is_sandbox
  AND (rr.country IS NULL OR rr.country = cp.country)
  AND (rr.currency IS NULL OR rr.currency = cp.currency)
LEFT JOIN requisites r ON r.provider_id = p.id
  AND r.status = 'ACTIVE'
  AND r.currency = cp.currency
  AND r.country = cp.country
LEFT JOIN disputes d ON d.provider_id = p.id
  AND d.status IN ('NEW', 'UNDER_REVIEW', 'AWAITING_PROVIDER_RESPONSE')
WHERE p.id = cp.provider_id
GROUP BY p.id, p.name, p.status, p.traffic_enabled, p.traffic_disabled_reason;

-- РЕШЕНИЯ ПРОБЛЕМ:

-- Если status != 'ACTIVE':
UPDATE providers SET status = 'ACTIVE', updated_at = NOW() WHERE id = 'YOUR_PROVIDER_ID';

-- Если traffic_enabled = false:
UPDATE providers
SET traffic_enabled = true,
    traffic_disabled_reason = NULL,
    traffic_disabled_at = NULL,
    traffic_disabled_by = NULL,
    updated_at = NOW()
WHERE id = 'YOUR_PROVIDER_ID';

-- Если нет правил маршрутизации, создать:
INSERT INTO route_rules (id, priority, weight, provider_id, status, is_sandbox, created_at, updated_at)
VALUES (gen_random_uuid(), 1, 100, 'YOUR_PROVIDER_ID', 'ACTIVE', false, NOW(), NOW());

-- Если есть активные споры - закрыть или разрешить их:
UPDATE disputes
SET status = 'CLOSED', resolved_at = NOW(), updated_at = NOW()
WHERE provider_id = 'YOUR_PROVIDER_ID'
  AND status IN ('NEW', 'UNDER_REVIEW', 'AWAITING_PROVIDER_RESPONSE');

-- После закрытия споров включить трафик вручную:
INSERT INTO provider_traffic_history (id, provider_id, action, reason, created_at)
VALUES (gen_random_uuid(), 'YOUR_PROVIDER_ID', 'ENABLED', 'Спор разрешен, трафик восстановлен', NOW());
