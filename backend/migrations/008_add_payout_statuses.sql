-- Add payout statuses to transaction status constraint
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_status_check;

ALTER TABLE transactions ADD CONSTRAINT transactions_status_check
CHECK (status IN ('NEW', 'ASSIGNED', 'WAITING_PAYMENT', 'PAID', 'EXPIRED', 'CANCELLED', 'PAYOUT_SUCCESS', 'PAYOUT_ERROR'));
