-- GAP-DATA-01: replace raw IP/UA/User-ID columns with versioned HighwayHash fields.

USE ad_event_processor;

DROP VIEW IF EXISTS mv_ml_features_1m_impressions;
DROP VIEW IF EXISTS mv_ml_features_1m_clicks;

ALTER TABLE impressions ADD COLUMN IF NOT EXISTS ip_hash FixedString(16) DEFAULT '';
ALTER TABLE impressions ADD COLUMN IF NOT EXISTS ua_hash FixedString(16) DEFAULT '';
ALTER TABLE impressions ADD COLUMN IF NOT EXISTS pii_salt_version UInt8 DEFAULT 1;
ALTER TABLE impressions DROP COLUMN IF EXISTS ip_address;
ALTER TABLE impressions DROP COLUMN IF EXISTS user_agent;

ALTER TABLE clicks ADD COLUMN IF NOT EXISTS ip_hash FixedString(16) DEFAULT '';
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS ua_hash FixedString(16) DEFAULT '';
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS pii_salt_version UInt8 DEFAULT 1;
ALTER TABLE clicks DROP COLUMN IF EXISTS ip_address;
ALTER TABLE clicks DROP COLUMN IF EXISTS user_agent;

ALTER TABLE conversions ADD COLUMN IF NOT EXISTS ip_hash FixedString(16) DEFAULT '';
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS ua_hash FixedString(16) DEFAULT '';
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS pii_salt_version UInt8 DEFAULT 1;
ALTER TABLE conversions DROP COLUMN IF EXISTS ip_address;
ALTER TABLE conversions DROP COLUMN IF EXISTS user_agent;

ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS ip_hash FixedString(16) DEFAULT '';
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS ua_hash FixedString(16) DEFAULT '';
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS user_id_hash FixedString(16) DEFAULT '';
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS pii_salt_version UInt8 DEFAULT 1;
ALTER TABLE fraud_events DROP COLUMN IF EXISTS ip_address;
ALTER TABLE fraud_events DROP COLUMN IF EXISTS user_agent;
ALTER TABLE fraud_events DROP COLUMN IF EXISTS user_id;

ALTER TABLE fraud_aggregate_spikes ADD COLUMN IF NOT EXISTS subnet_hash FixedString(16) DEFAULT '';
ALTER TABLE fraud_aggregate_spikes DROP COLUMN IF EXISTS subnet;

ALTER TABLE ml_features_1m ADD COLUMN IF NOT EXISTS ip_hash FixedString(16) DEFAULT '';
ALTER TABLE ml_features_1m DROP COLUMN IF EXISTS ip_address;

ALTER TABLE ml_shadow_scores ADD COLUMN IF NOT EXISTS ip_hash FixedString(16) DEFAULT '';
ALTER TABLE ml_shadow_scores DROP COLUMN IF EXISTS ip_address;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_ml_features_1m_impressions
TO ml_features_1m
AS SELECT
    toStartOfMinute(created_at) AS window_start,
    ip_hash,
    campaign_id,
    count() AS events,
    toUInt64(0) AS clicks,
    toInt64(0) AS spend_micro,
    toInt64(0) AS budget_limit_micro,
    uniqCombined(ip_hash) AS unique_users,
    uniqCombined(ua_hash) AS unique_uas
FROM impressions
GROUP BY window_start, ip_hash, campaign_id;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_ml_features_1m_clicks
TO ml_features_1m
AS SELECT
    toStartOfMinute(created_at) AS window_start,
    ip_hash,
    campaign_id,
    count() AS events,
    count() AS clicks,
    toInt64(0) AS spend_micro,
    toInt64(0) AS budget_limit_micro,
    uniqCombined(ip_hash) AS unique_users,
    uniqCombined(ua_hash) AS unique_uas
FROM clicks
GROUP BY window_start, ip_hash, campaign_id;
