USE ad_event_processor;

ALTER TABLE tg_events_raw ADD COLUMN IF NOT EXISTS event_type LowCardinality(String) DEFAULT '';
ALTER TABLE tg_events ADD COLUMN IF NOT EXISTS event_type LowCardinality(String) DEFAULT '';

DROP TABLE IF EXISTS mv_tg_events_raw_to_tg_events;

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
    created_at,
    event_type
FROM tg_events_raw;
