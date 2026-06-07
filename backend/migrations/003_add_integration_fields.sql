-- Add integration fields for providers and casinos
ALTER TABLE providers ADD COLUMN IF NOT EXISTS merchant_id VARCHAR(255);
ALTER TABLE providers ADD COLUMN IF NOT EXISTS base_url TEXT;

ALTER TABLE casinos ADD COLUMN IF NOT EXISTS merchant_id VARCHAR(255);
ALTER TABLE casinos ADD COLUMN IF NOT EXISTS base_url TEXT;
ALTER TABLE casinos ADD COLUMN IF NOT EXISTS secret_key VARCHAR(128);

-- Create indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_providers_merchant_id ON providers(merchant_id);
CREATE INDEX IF NOT EXISTS idx_casinos_merchant_id ON casinos(merchant_id);
