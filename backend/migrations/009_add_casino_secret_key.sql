-- Add secret_key field to casinos table for webhook signature generation
ALTER TABLE casinos ADD COLUMN IF NOT EXISTS secret_key VARCHAR(128);

-- Generate secret keys for existing casinos
UPDATE casinos SET secret_key = encode(gen_random_bytes(32), 'hex') WHERE secret_key IS NULL;
