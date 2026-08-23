CREATE TABLE IF NOT EXISTS ad_event_processor.residential_intel_cache (
    ip_hash FixedString(16),
    residential_proxy UInt8,
    vpn UInt8,
    proxy UInt8,
    provider LowCardinality(String),
    cached_at DateTime64(3, 'UTC')
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(cached_at)
ORDER BY (cached_at, ip_hash)
TTL toDateTime(cached_at) + INTERVAL 90 DAY;
