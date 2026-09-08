USE ad_event_processor;

ALTER TABLE clicks ADD COLUMN IF NOT EXISTS in_app_webview_class UInt8 DEFAULT 0;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS in_app_webview_class UInt8 DEFAULT 0;
