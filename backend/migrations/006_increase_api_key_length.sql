-- Increase api_key length to support longer keys (pk_ + 64 hex chars = 67 chars)
ALTER TABLE providers ALTER COLUMN api_key TYPE VARCHAR(128);
ALTER TABLE casinos ALTER COLUMN api_key TYPE VARCHAR(128);
