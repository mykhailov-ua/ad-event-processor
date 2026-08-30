-- ClickHouse bootstrap DDL (deploy/clickhouse/init.sql).
-- Must match internal/clickhouse/migrate/00000_bootstrap_tables.sql in the same commit.
-- Cold path only: processor ingests streams; IVT/ML/reports read these tables.
CREATE DATABASE IF NOT EXISTS ad_event_processor;
USE ad_event_processor;

-- Core event facts: impressions, clicks, conversions (ReplacingMergeTree, 180d TTL).
-- ReplacingMergeTree(created_at) deduplicates duplicate tracking deliveries.
-- Partitioned by month; ORDER BY (campaign_id, created_at, click_id).
CREATE TABLE IF NOT EXISTS impressions (
    click_id String,
    campaign_id UUID,
    placement_id String DEFAULT '',
    ip_hash FixedString(16),
    ua_hash FixedString(16),
    pii_salt_version UInt8 DEFAULT 1,
    payload String,
    created_at DateTime64(3, 'UTC')
) ENGINE = ReplacingMergeTree(created_at)
PARTITION BY toYYYYMM(created_at)
ORDER BY (campaign_id, created_at, click_id)
TTL toDateTime(created_at) + INTERVAL 180 DAY;

-- Click and conversion facts (same engine/TTL as impressions).
CREATE TABLE IF NOT EXISTS clicks (
    click_id String,
    campaign_id UUID,
    placement_id String DEFAULT '',
    ip_hash FixedString(16),
    ua_hash FixedString(16),
    pii_salt_version UInt8 DEFAULT 1,
    tls_hash String DEFAULT '',
    payload String,
    created_at DateTime64(3, 'UTC')
) ENGINE = ReplacingMergeTree(created_at)
PARTITION BY toYYYYMM(created_at)
ORDER BY (campaign_id, created_at, click_id)
TTL toDateTime(created_at) + INTERVAL 180 DAY;

CREATE TABLE IF NOT EXISTS conversions (
    click_id String,
    campaign_id UUID,
    placement_id String DEFAULT '',
    ip_hash FixedString(16),
    ua_hash FixedString(16),
    pii_salt_version UInt8 DEFAULT 1,
    payload String,
    created_at DateTime64(3, 'UTC')
) ENGINE = ReplacingMergeTree(created_at)
PARTITION BY toYYYYMM(created_at)
ORDER BY (campaign_id, created_at, click_id)
TTL toDateTime(created_at) + INTERVAL 180 DAY;

-- Fraud stream rows: reject reasons, fraud_score, silent_reject_event flag (90d TTL).
CREATE TABLE IF NOT EXISTS fraud_events (
    click_id String,
    campaign_id UUID,
    user_id_hash FixedString(16),
    event_type String,
    ip_hash FixedString(16),
    ua_hash FixedString(16),
    pii_salt_version UInt8 DEFAULT 1,
    payload String,
    fraud_reason String,
    fraud_score UInt32 DEFAULT 0,
    silent_reject_event UInt8 DEFAULT 0,
    layer_desync_count UInt8 DEFAULT 0,
    created_at DateTime64(3, 'UTC')
) ENGINE = ReplacingMergeTree(created_at)
PARTITION BY toYYYYMM(created_at)
ORDER BY (campaign_id, created_at, click_id)
TTL toDateTime(created_at) + INTERVAL 90 DAY;

-- Subnet-level fraud spike aggregates for IVT batch jobs (SummingMergeTree, 90d TTL).
CREATE TABLE IF NOT EXISTS fraud_aggregate_spikes (
    subnet_hash FixedString(16),
    ipv6_prefix String DEFAULT '',
    fraud_reason LowCardinality(String),
    event_count UInt64,
    window_ms UInt32,
    created_at DateTime64(3, 'UTC')
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (subnet_hash, fraud_reason, created_at)
    TTL toDateTime(created_at) + INTERVAL 90 DAY;

-- Residential/VPN/proxy intel cache keyed by ip_hash (MergeTree, 90d TTL).
CREATE TABLE IF NOT EXISTS residential_intel_cache (
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

-- Hourly audit rollups for warm-storage export and compliance retention (365d TTL).
CREATE TABLE IF NOT EXISTS audit_log_rollups (
    rollup_hour DateTime('UTC'),
    campaign_id UUID,
    event_type LowCardinality(String),
    event_count UInt64,
    fraud_event_count UInt64,
    billable_event_count UInt64,
    sample_click_ids Array(String),
    source_segment String,
    warm_dest_sha256 String,
    ingested_at DateTime64(3, 'UTC') DEFAULT now64(3)
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(rollup_hour)
ORDER BY (campaign_id, rollup_hour, event_type, source_segment, warm_dest_sha256)
TTL rollup_hour + INTERVAL 365 DAY;

-- Filter reject counters by kind (SummingMergeTree, 90d TTL).
CREATE TABLE IF NOT EXISTS filter_reject_rollups (
    rollup_hour DateTime('UTC'),
    reject_kind LowCardinality(String),
    reject_count UInt64
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(rollup_hour)
ORDER BY (rollup_hour, reject_kind)
TTL rollup_hour + INTERVAL 90 DAY;

-- Filter reject slices by placement and country (SummingMergeTree, 90d TTL).
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

-- Edge timing columns (ms): rtt_syn_ms, ttfb_app_ms, rtt_split_delta_ms from nginx/XDP headers.
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS rtt_syn_ms UInt16 DEFAULT 0;
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS ttfb_app_ms UInt16 DEFAULT 0;
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS rtt_split_delta_ms UInt16 DEFAULT 0;

ALTER TABLE conversions ADD COLUMN IF NOT EXISTS rtt_syn_ms UInt16 DEFAULT 0;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS ttfb_app_ms UInt16 DEFAULT 0;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS rtt_split_delta_ms UInt16 DEFAULT 0;

ALTER TABLE impressions ADD COLUMN IF NOT EXISTS rtt_syn_ms UInt16 DEFAULT 0;
ALTER TABLE impressions ADD COLUMN IF NOT EXISTS ttfb_app_ms UInt16 DEFAULT 0;
ALTER TABLE impressions ADD COLUMN IF NOT EXISTS rtt_split_delta_ms UInt16 DEFAULT 0;

-- Mobile biometric and device_type columns on conversions (cold-path enrichment).
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS mobile_touch_count UInt8 DEFAULT 0;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS mobile_gyro_samples UInt8 DEFAULT 0;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS mobile_gyro_variance UInt16 DEFAULT 0;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS mobile_gyro_flat UInt8 DEFAULT 0;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS mobile_biometric_set UInt8 DEFAULT 0;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS mobile_biometric_mobile UInt8 DEFAULT 0;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS device_type LowCardinality(String) DEFAULT '';
