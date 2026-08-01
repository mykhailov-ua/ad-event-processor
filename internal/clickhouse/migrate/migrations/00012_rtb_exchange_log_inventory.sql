ALTER TABLE rtb_exchange_log
    ADD COLUMN IF NOT EXISTS inventory LowCardinality(String) DEFAULT '',
    ADD COLUMN IF NOT EXISTS geo_country FixedString(2) DEFAULT '',
    ADD COLUMN IF NOT EXISTS device_os LowCardinality(String) DEFAULT '',
    ADD COLUMN IF NOT EXISTS media_w UInt16 DEFAULT 0,
    ADD COLUMN IF NOT EXISTS media_h UInt16 DEFAULT 0;
