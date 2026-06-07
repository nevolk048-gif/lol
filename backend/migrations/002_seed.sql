-- Seed data for PaymentsGate

-- Password: Admin123! (bcrypt hash)
INSERT INTO users (id, email, password_hash, role, status) VALUES
('a0000000-0000-0000-0000-000000000001', 'admin@paymentsgate.io', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'SUPER_ADMIN', 'ACTIVE'),
('a0000000-0000-0000-0000-000000000002', 'support@paymentsgate.io', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'SUPPORT', 'ACTIVE'),
('a0000000-0000-0000-0000-000000000003', 'analyst@paymentsgate.io', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'ANALYST', 'ACTIVE');

INSERT INTO providers (id, name, api_key, secret_key, webhook_url, status, is_sandbox) VALUES
('b0000000-0000-0000-0000-000000000001', 'PayFlow Global', 'pk_prod_payflow001', 'sk_prod_payflow001secret', 'https://api.payflow.io/webhook', 'ACTIVE', false),
('b0000000-0000-0000-0000-000000000002', 'SwiftPay EU', 'pk_prod_swiftpay001', 'sk_prod_swiftpay001secret', 'https://api.swiftpay.eu/webhook', 'ACTIVE', false),
('b0000000-0000-0000-0000-000000000003', 'AsiaPay Connect', 'pk_prod_asiapay001', 'sk_prod_asiapay001secret', 'https://api.asiapay.com/webhook', 'ACTIVE', false),
('b0000000-0000-0000-0000-000000000004', 'Sandbox Provider', 'pk_sandbox_provider', 'sk_sandbox_provider_secret', NULL, 'ACTIVE', true);

INSERT INTO casinos (id, name, api_key, webhook_url, status, is_sandbox) VALUES
('c0000000-0000-0000-0000-000000000001', 'Royal Casino', 'pk_casino_royal001', 'https://royal-casino.com/webhook', 'ACTIVE', false),
('c0000000-0000-0000-0000-000000000002', 'Diamond Slots', 'pk_casino_diamond001', 'https://diamond-slots.com/webhook', 'ACTIVE', false),
('c0000000-0000-0000-0000-000000000003', 'Golden Palace', 'pk_casino_golden001', 'https://golden-palace.com/webhook', 'ACTIVE', false),
('c0000000-0000-0000-0000-000000000004', 'Sandbox Casino', 'pk_sandbox_casino', NULL, 'ACTIVE', true);

INSERT INTO requisites (id, provider_id, bank_name, holder_name, account_number, currency, country, daily_limit, used_limit, status, is_sandbox) VALUES
('d0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001', 'Chase Bank', 'PayFlow Holdings LLC', '****4521', 'USD', 'US', 500000, 125000, 'ACTIVE', false),
('d0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000001', 'Bank of America', 'PayFlow Holdings LLC', '****7832', 'USD', 'US', 300000, 89000, 'ACTIVE', false),
('d0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000002', 'Deutsche Bank', 'SwiftPay GmbH', '****9012', 'EUR', 'DE', 400000, 156000, 'ACTIVE', false),
('d0000000-0000-0000-0000-000000000004', 'b0000000-0000-0000-0000-000000000002', 'Barclays', 'SwiftPay UK Ltd', '****3344', 'GBP', 'GB', 250000, 67000, 'ACTIVE', false),
('d0000000-0000-0000-0000-000000000005', 'b0000000-0000-0000-0000-000000000003', 'DBS Bank', 'AsiaPay Pte Ltd', '****5566', 'USD', 'SG', 600000, 234000, 'ACTIVE', false),
('d0000000-0000-0000-0000-000000000006', 'b0000000-0000-0000-0000-000000000004', 'Sandbox Bank', 'Test Account', 'SB-00000001', 'USD', 'US', 1000000, 0, 'ACTIVE', true);

INSERT INTO route_rules (id, priority, weight, country, currency, provider_id, status, is_fallback, is_sandbox) VALUES
('e0000000-0000-0000-0000-000000000001', 1, 100, 'US', 'USD', 'b0000000-0000-0000-0000-000000000001', 'ACTIVE', false, false),
('e0000000-0000-0000-0000-000000000002', 2, 80, 'DE', 'EUR', 'b0000000-0000-0000-0000-000000000002', 'ACTIVE', false, false),
('e0000000-0000-0000-0000-000000000003', 2, 80, 'GB', 'GBP', 'b0000000-0000-0000-0000-000000000002', 'ACTIVE', false, false),
('e0000000-0000-0000-0000-000000000004', 3, 60, NULL, 'USD', 'b0000000-0000-0000-0000-000000000003', 'ACTIVE', false, false),
('e0000000-0000-0000-0000-000000000005', 99, 50, NULL, NULL, 'b0000000-0000-0000-0000-000000000001', 'ACTIVE', true, false),
('e0000000-0000-0000-0000-000000000006', 1, 100, NULL, NULL, 'b0000000-0000-0000-0000-000000000004', 'ACTIVE', false, true);

-- Sample transactions (last 30 days)
INSERT INTO transactions (id, external_id, casino_id, provider_id, requisite_id, amount, currency, country, status, is_sandbox, processing_ms, created_at, assigned_at, paid_at) VALUES
('f0000000-0000-0000-0000-000000000001', 'EXT-001', 'c0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000001', 1500.00, 'USD', 'US', 'PAID', false, 245, NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1 hour'),
('f0000000-0000-0000-0000-000000000002', 'EXT-002', 'c0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000003', 750.00, 'EUR', 'DE', 'PAID', false, 189, NOW() - INTERVAL '5 hours', NOW() - INTERVAL '5 hours', NOW() - INTERVAL '4 hours'),
('f0000000-0000-0000-0000-000000000003', 'EXT-003', 'c0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000002', 2500.00, 'USD', 'US', 'WAITING_PAYMENT', false, 312, NOW() - INTERVAL '30 minutes', NOW() - INTERVAL '30 minutes', NULL),
('f0000000-0000-0000-0000-000000000004', 'EXT-004', 'c0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000003', 'd0000000-0000-0000-0000-000000000005', 500.00, 'USD', 'SG', 'PAID', false, 156, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day', NOW() - INTERVAL '23 hours'),
('f0000000-0000-0000-0000-000000000005', 'EXT-005', 'c0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000004', 320.00, 'GBP', 'GB', 'EXPIRED', false, 278, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days', NULL),
('f0000000-0000-0000-0000-000000000006', 'EXT-006', 'c0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000001', 10000.00, 'USD', 'US', 'PAID', false, 198, NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days' + INTERVAL '15 minutes'),
('f0000000-0000-0000-0000-000000000007', 'EXT-007', 'c0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000003', 1800.00, 'EUR', 'DE', 'PAID', false, 223, NOW() - INTERVAL '6 hours', NOW() - INTERVAL '6 hours', NOW() - INTERVAL '5 hours'),
('f0000000-0000-0000-0000-000000000008', 'EXT-008', 'c0000000-0000-0000-0000-000000000004', 'b0000000-0000-0000-0000-000000000004', 'd0000000-0000-0000-0000-000000000006', 100.00, 'USD', 'US', 'PAID', true, 145, NOW() - INTERVAL '1 hour', NOW() - INTERVAL '1 hour', NOW() - INTERVAL '45 minutes');

INSERT INTO audit_logs (user_id, action, entity_type, entity_id, ip_address, details) VALUES
('a0000000-0000-0000-0000-000000000001', 'LOGIN', 'user', 'a0000000-0000-0000-0000-000000000001', '192.168.1.1', '{"browser":"Chrome"}'),
('a0000000-0000-0000-0000-000000000001', 'CREATE', 'provider', 'b0000000-0000-0000-0000-000000000001', '192.168.1.1', '{"name":"PayFlow Global"}'),
('a0000000-0000-0000-0000-000000000001', 'UPDATE', 'route_rule', 'e0000000-0000-0000-0000-000000000001', '192.168.1.1', '{"weight":100}'),
(NULL, 'TRANSACTION_ROUTED', 'transaction', 'f0000000-0000-0000-0000-000000000001', '', '{}');

INSERT INTO integration_logs (endpoint, method, status_code, duration_ms, casino_id, transaction_id, is_sandbox) VALUES
('/api/v1/deposit/create', 'POST', 201, 245, 'c0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000001', false),
('/api/v1/deposit/status/f0000000-0000-0000-0000-000000000001', 'GET', 200, 12, 'c0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000001', false),
('/api/v1/deposit/create', 'POST', 201, 189, 'c0000000-0000-0000-0000-000000000002', 'f0000000-0000-0000-0000-000000000002', false),
('/api/v1/deposit/create', 'POST', 400, 5, 'c0000000-0000-0000-0000-000000000003', NULL, false);
