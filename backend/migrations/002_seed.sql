-- Seed data for PaymentsGate

-- Default super admin account
-- Email: admin@paymentsgate.io
-- Password: Admin123!
INSERT INTO users (id, email, password_hash, role, status) VALUES
('a0000000-0000-0000-0000-000000000001', 'admin@paymentsgate.io', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'SUPER_ADMIN', 'ACTIVE')
ON CONFLICT (id) DO NOTHING;
