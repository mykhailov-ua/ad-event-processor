package stream

func ClickHouseSpoolConfigFromConfig(segmentMB, maxSegments int) ClickHouseSpoolConfig {
	cfg := DefaultClickHouseSpoolConfig()
	if segmentMB > 0 {
		cfg.SegmentSizeBytes = int64(segmentMB) * 1024 * 1024
	}
	if maxSegments > 0 {
		cfg.MaxSegments = maxSegments
	}
	return cfg
}
