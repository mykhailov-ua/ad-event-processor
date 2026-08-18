ALTER TABLE rtb_exchange_log
    ADD COLUMN IF NOT EXISTS source_tid LowCardinality(String) DEFAULT '',
    ADD COLUMN IF NOT EXISTS connection_type UInt8 DEFAULT 0,
    ADD COLUMN IF NOT EXISTS pmp_private UInt8 DEFAULT 0,
    ADD COLUMN IF NOT EXISTS device_lmt UInt8 DEFAULT 0,
    ADD COLUMN IF NOT EXISTS viewability_ppm UInt32 DEFAULT 0,
    ADD COLUMN IF NOT EXISTS eid_source LowCardinality(String) DEFAULT '',
    ADD COLUMN IF NOT EXISTS app_ver LowCardinality(String) DEFAULT '';
