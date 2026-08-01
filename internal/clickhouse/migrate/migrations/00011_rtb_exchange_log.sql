
CREATE TABLE IF NOT EXISTS rtb_exchange_log (
    request_id String,
    bid_id String,
    won UInt8,
    no_bid_reason UInt16,
    price_micro Int64,
    deal_id LowCardinality(String),
    created_at DateTime64(3, 'UTC')
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (request_id, created_at)
TTL toDateTime(created_at) + INTERVAL 30 DAY;
