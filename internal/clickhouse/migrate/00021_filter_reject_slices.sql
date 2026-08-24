USE ad_event_processor;

CREATE TABLE IF NOT EXISTS filter_reject_slices (
    rollup_hour DateTime('UTC'),
    reject_kind LowCardinality(String),
    placement_id String DEFAULT '',
    country FixedString(2) DEFAULT '',
    reject_count UInt64
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(rollup_hour)
ORDER BY (rollup_hour, reject_kind, country, placement_id)
TTL rollup_hour + INTERVAL 90 DAY;
