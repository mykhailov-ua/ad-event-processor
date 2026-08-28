ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS layer_desync_count UInt8 DEFAULT 0;
