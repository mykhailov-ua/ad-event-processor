USE ad_event_processor;

ALTER TABLE conversions ADD COLUMN IF NOT EXISTS mobile_touch_count UInt8 DEFAULT 0;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS mobile_gyro_samples UInt8 DEFAULT 0;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS mobile_gyro_variance UInt16 DEFAULT 0;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS mobile_gyro_flat UInt8 DEFAULT 0;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS mobile_biometric_set UInt8 DEFAULT 0;
ALTER TABLE conversions ADD COLUMN IF NOT EXISTS mobile_biometric_mobile UInt8 DEFAULT 0;
