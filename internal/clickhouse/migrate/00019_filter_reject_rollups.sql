USE ad_event_processor;

CREATE TABLE IF NOT EXISTS filter_reject_rollups (
    rollup_hour DateTime('UTC'),
    reject_kind LowCardinality(String),
    reject_count UInt64
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(rollup_hour)
ORDER BY (rollup_hour, reject_kind)
TTL rollup_hour + INTERVAL 90 DAY;
