-- Update Demo Casino webhook URL to Vercel domain
UPDATE casinos
SET
    webhook_url = 'https://merchant-demo-pi.vercel.app/api/webhooks/paymentsgate'
WHERE name = 'Demo Casino';

-- Verify the update
SELECT
    id,
    name,
    webhook_url,
    secret_key,
    status
FROM casinos
WHERE name = 'Demo Casino';
