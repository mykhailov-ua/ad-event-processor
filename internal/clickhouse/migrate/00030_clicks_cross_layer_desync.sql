USE ad_event_processor;

ALTER TABLE clicks ADD COLUMN IF NOT EXISTS cross_layer_desync_fired UInt8 DEFAULT 0;
