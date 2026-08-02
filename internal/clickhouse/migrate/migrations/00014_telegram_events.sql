
USE ad_event_processor;

CREATE TABLE IF NOT EXISTS tg_events_raw (
    click_id String,
    campaign_id UUID,
    tg_user_id String,
    start_param String,
    chat_type String,
    is_premium UInt8,
    motivated UInt8,
    widget_id String,
    bot_id UInt64,
    payload String,
    created_at DateTime64(3, 'UTC')
) ENGINE = ReplacingMergeTree(created_at)
PARTITION BY toYYYYMM(created_at)
ORDER BY (campaign_id, created_at, click_id)
TTL toDateTime(created_at) + INTERVAL 3 DAY;

CREATE TABLE IF NOT EXISTS tg_events (
    click_id String,
    campaign_id UUID,
    tg_user_id_sha256 FixedString(64),
    start_param String,
    chat_type String,
    is_premium UInt8,
    motivated UInt8,
    widget_id String,
    bot_id UInt64,
    payload String,
    created_at DateTime64(3, 'UTC')
) ENGINE = ReplacingMergeTree(created_at)
PARTITION BY toYYYYMM(created_at)
ORDER BY (campaign_id, created_at, click_id)
TTL toDateTime(created_at) + INTERVAL 180 DAY;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_tg_events_raw_to_tg_events
TO tg_events
AS SELECT
    click_id,
    campaign_id,
    lower(hex(SHA256(tg_user_id))) AS tg_user_id_sha256,
    start_param,
    chat_type,
    is_premium,
    motivated,
    widget_id,
    bot_id,
    payload,
    created_at
FROM tg_events_raw;
