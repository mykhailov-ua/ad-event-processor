USE ad_event_processor;

CREATE TABLE IF NOT EXISTS cost_attribution_runs (
    sync_run_id UUID,
    click_id String,
    attributed_cost_micro Int64,
    cost_source LowCardinality(String),
    applied_at DateTime DEFAULT now()
) ENGINE = ReplacingMergeTree(applied_at)
ORDER BY (sync_run_id, click_id);
