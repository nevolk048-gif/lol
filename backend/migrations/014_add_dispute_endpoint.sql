-- Configurable per-provider dispute endpoint (path appended to base_url).
-- MajorPay base_url is https://api.majorpay.io/api and payments work at
-- {base_url}/payments, so the dispute endpoint is {base_url}/dispute.
-- The previous hardcoded "/disputes" produced .../api/disputes -> 404.
ALTER TABLE providers ADD COLUMN IF NOT EXISTS dispute_endpoint VARCHAR(255) NOT NULL DEFAULT '/dispute';

-- Ensure MajorPay uses the correct path (relative to base_url which already ends with /api).
UPDATE providers SET dispute_endpoint = '/dispute' WHERE name = 'MajorPay';
