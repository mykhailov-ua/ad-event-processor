USE ad_event_processor;

ALTER TABLE impressions ADD COLUMN IF NOT EXISTS placement_id String AFTER campaign_id;
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS placement_id String AFTER campaign_id;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS placement_id String AFTER campaign_id;


CREATE TABLE IF NOT EXISTS placement_stats_hourly (
    campaign_id UUID,
    placement_id String,
    hour DateTime,
    spend_micro Int64,
    revenue_micro Int64,
    click_count UInt64,
    conversion_count UInt64
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (campaign_id, placement_id, hour);


CREATE MATERIALIZED VIEW IF NOT EXISTS mv_placement_stats_money_hourly
TO placement_stats_hourly
AS SELECT
    campaign_id,
    placement_id,
    toStartOfHour(snapshot_hour) AS hour,
    sumIf(amount_usd_micro, line_type = 'spend') AS spend_micro,
    sumIf(amount_usd_micro, line_type = 'revenue') AS revenue_micro,
    0 AS click_count,
    0 AS conversion_count
FROM cost_snapshots
GROUP BY campaign_id, placement_id, hour;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_placement_stats_clicks_hourly
TO placement_stats_hourly
AS SELECT
    campaign_id,
    placement_id,
    toStartOfHour(created_at) AS hour,
    0 AS spend_micro,
    0 AS revenue_micro,
    count() AS click_count,
    0 AS conversion_count
FROM clicks
GROUP BY campaign_id, placement_id, hour;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_placement_stats_convs_hourly
TO placement_stats_hourly
AS SELECT
    campaign_id,
    placement_id,
    toStartOfHour(created_at) AS hour,
    0 AS spend_micro,
    0 AS revenue_micro,
    0 AS click_count,
    count() AS conversion_count
FROM conversions
GROUP BY campaign_id, placement_id, hour;

CREATE OR REPLACE VIEW mv_placement_stats_hourly AS SELECT * FROM placement_stats_hourly;
