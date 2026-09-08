USE ad_event_processor;

ALTER TABLE clicks ADD COLUMN IF NOT EXISTS probe_behavior_score UInt8 DEFAULT 0;
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS footer_reach_ms UInt32 DEFAULT 0;
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS event_order_entropy UInt16 DEFAULT 0;

ALTER TABLE conversions ADD COLUMN IF NOT EXISTS probe_behavior_score UInt8 DEFAULT 0;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS footer_reach_ms UInt32 DEFAULT 0;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS event_order_entropy UInt16 DEFAULT 0;
