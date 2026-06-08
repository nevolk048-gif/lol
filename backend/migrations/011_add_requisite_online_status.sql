-- Add online status tracking for requisites
ALTER TABLE requisites ADD COLUMN IF NOT EXISTS is_online BOOLEAN DEFAULT true;
ALTER TABLE requisites ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ;

-- Add index for online requisites lookup
CREATE INDEX IF NOT EXISTS idx_requisites_online ON requisites(is_online, status) WHERE status = 'ACTIVE';

COMMENT ON COLUMN requisites.is_online IS 'Whether the trader/requisite is currently online and accepting payments';
COMMENT ON COLUMN requisites.last_seen_at IS 'Last time the trader was seen online (updated via provider API or manual check)';
