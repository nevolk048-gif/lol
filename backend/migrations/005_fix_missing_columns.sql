-- Fix missing columns in providers and casinos tables
-- This migration ensures all required columns exist

-- Add missing columns to providers if they don't exist
ALTER TABLE providers ADD COLUMN IF NOT EXISTS merchant_id VARCHAR(255);
ALTER TABLE providers ADD COLUMN IF NOT EXISTS base_url TEXT;

-- Add missing columns to casinos if they don't exist
ALTER TABLE casinos ADD COLUMN IF NOT EXISTS merchant_id VARCHAR(255);
ALTER TABLE casinos ADD COLUMN IF NOT EXISTS base_url TEXT;
ALTER TABLE casinos ADD COLUMN IF NOT EXISTS secret_key VARCHAR(128);

-- Create indexes if they don't exist
CREATE INDEX IF NOT EXISTS idx_providers_merchant_id ON providers(merchant_id);
CREATE INDEX IF NOT EXISTS idx_casinos_merchant_id ON casinos(merchant_id);

-- Verify columns exist
DO $$
BEGIN
    -- Check providers table
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'providers' AND column_name = 'merchant_id'
    ) THEN
        RAISE EXCEPTION 'Column merchant_id missing from providers table';
    END IF;

    -- Check casinos table
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'casinos' AND column_name = 'secret_key'
    ) THEN
        RAISE EXCEPTION 'Column secret_key missing from casinos table';
    END IF;

    RAISE NOTICE 'All required columns exist';
END $$;
