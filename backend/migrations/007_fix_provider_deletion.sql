-- Fix foreign key constraints to allow provider deletion
-- Change provider_id in transactions to SET NULL on delete instead of preventing deletion
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_provider_id_fkey;
ALTER TABLE transactions ADD CONSTRAINT transactions_provider_id_fkey
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE SET NULL;

-- Do the same for casino_id if needed in the future
-- For now, keeping it as is since casinos are less likely to be deleted
