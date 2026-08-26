USE ad_event_processor;

ALTER TABLE clicks ADD COLUMN IF NOT EXISTS attributed_cost_micro Int64 DEFAULT 0;
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS cost_source LowCardinality(String) DEFAULT '';
