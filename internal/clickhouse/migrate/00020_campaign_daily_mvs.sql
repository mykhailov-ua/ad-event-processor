USE ad_event_processor;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_campaign_daily_impressions
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(day)
ORDER BY (campaign_id, day)
POPULATE
AS SELECT
    campaign_id,
    toDate(created_at) AS day,
    count() AS impression_count
FROM impressions
GROUP BY campaign_id, day;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_campaign_daily_clicks
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(day)
ORDER BY (campaign_id, day)
POPULATE
AS SELECT
    campaign_id,
    toDate(created_at) AS day,
    count() AS click_count
FROM clicks
GROUP BY campaign_id, day;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_campaign_daily_conversions
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(day)
ORDER BY (campaign_id, day)
POPULATE
AS SELECT
    campaign_id,
    toDate(created_at) AS day,
    count() AS conversion_count
FROM conversions
GROUP BY campaign_id, day;
