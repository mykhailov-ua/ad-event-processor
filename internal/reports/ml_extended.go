package reports

type MLScoreDistributionReportResponse struct {
	Rows       []MLScoreBucketDTO `json:"rows"`
	Freshness  DataFreshnessDTO   `json:"freshness"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

type MLShadowDeltaReportResponse struct {
	Rows       []MLShadowDeltaRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO      `json:"freshness"`
	NextCursor string                `json:"next_cursor,omitempty"`
}

type MLFeatureSpikesReportResponse struct {
	Rows       []MLFeatureSpikeRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO       `json:"freshness"`
	NextCursor string                 `json:"next_cursor,omitempty"`
}

func MLScoreDistributionRowsFromMaps(rows []map[string]any) []MLScoreBucketDTO {
	out := make([]MLScoreBucketDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, MLScoreBucketDTO{
			ScoreBucket: mapRowFloat64(row, "score_bucket"),
			RowCount:    mapRowInt64(row, "row_count"),
		})
	}
	return out
}

func MLShadowDeltaRowsFromMaps(rows []map[string]any) []MLShadowDeltaRowDTO {
	out := make([]MLShadowDeltaRowDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, MLShadowDeltaRowDTO{
			Bucket:           mapRowString(row, "bucket"),
			AvgShadowScore:   mapRowFloat64(row, "avg_shadow_score"),
			ScoreCount:       mapRowInt64(row, "score_count"),
			AvgFeatureEvents: mapRowFloat64(row, "avg_feature_events"),
		})
	}
	return out
}

func MLFeatureSpikeRowsFromMaps(rows []map[string]any) []MLFeatureSpikeRowDTO {
	out := make([]MLFeatureSpikeRowDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, MLFeatureSpikeRowDTO{
			WindowStart: mapRowString(row, "window_start"),
			Events:      mapRowInt64(row, "events"),
			Clicks:      mapRowInt64(row, "clicks"),
			Campaigns:   mapRowInt64(row, "campaigns"),
		})
	}
	return out
}
