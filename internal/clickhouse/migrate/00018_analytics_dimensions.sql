USE ad_event_processor;

ALTER TABLE impressions ADD COLUMN IF NOT EXISTS sub1 String DEFAULT '';
ALTER TABLE impressions ADD COLUMN IF NOT EXISTS sub2 String DEFAULT '';
ALTER TABLE impressions ADD COLUMN IF NOT EXISTS country FixedString(2) DEFAULT '';
ALTER TABLE impressions ADD COLUMN IF NOT EXISTS device_type LowCardinality(String) DEFAULT '';
ALTER TABLE impressions ADD COLUMN IF NOT EXISTS keyword String DEFAULT '';

ALTER TABLE clicks ADD COLUMN IF NOT EXISTS sub1 String DEFAULT '';
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS sub2 String DEFAULT '';
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS country FixedString(2) DEFAULT '';
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS device_type LowCardinality(String) DEFAULT '';
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS keyword String DEFAULT '';

ALTER TABLE conversions ADD COLUMN IF NOT EXISTS sub1 String DEFAULT '';
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS sub2 String DEFAULT '';
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS country FixedString(2) DEFAULT '';
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS device_type LowCardinality(String) DEFAULT '';
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS keyword String DEFAULT '';
