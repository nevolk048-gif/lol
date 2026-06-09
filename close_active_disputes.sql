-- Close all active disputes that are blocking provider routing
-- This will allow providers to start accepting transactions again

UPDATE disputes
SET status = 'RESOLVED',
    updated_at = NOW()
WHERE status IN ('NEW', 'UNDER_REVIEW', 'AWAITING_PROVIDER_RESPONSE');

-- Show the updated disputes
SELECT id, provider_id, status, reason, created_at, updated_at
FROM disputes
ORDER BY created_at DESC
LIMIT 10;
