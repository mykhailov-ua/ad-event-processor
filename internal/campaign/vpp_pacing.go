package campaign

import (
	"context"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
)

type VPPCampaignSample = reports.HourlyImpressionSample

func QueryVPPCampaignSamplesBatch(ctx context.Context, ch *database.ClickHouseQuery, from, to time.Time, campaignIDs []uuid.UUID) (map[uuid.UUID][]VPPCampaignSample, error) {
	if ch == nil || len(campaignIDs) == 0 {
		return map[uuid.UUID][]VPPCampaignSample{}, nil
	}
	clickhouseCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	out := make(map[uuid.UUID][]VPPCampaignSample, len(campaignIDs))
	for start := 0; start < len(campaignIDs); start += vppCampaignSampleBatch {
		end := start + vppCampaignSampleBatch
		if end > len(campaignIDs) {
			end = len(campaignIDs)
		}
		chunk, err := queryVPPCampaignSamplesChunk(clickhouseCtx, ch, from, to, campaignIDs[start:end])
		if err != nil {
			return nil, err
		}
		for id, samples := range chunk {
			out[id] = samples
		}
	}
	return out, nil
}

func queryVPPCampaignSamplesChunk(ctx context.Context, ch *database.ClickHouseQuery, from, to time.Time, campaignIDs []uuid.UUID) (map[uuid.UUID][]VPPCampaignSample, error) {
	query := `
SELECT campaign_id, toHour(hour) AS hr, sum(impression_count) AS impressions
FROM mv_campaign_hourly_impressions
WHERE hour >= ? AND hour < ? AND campaign_id IN (?)
GROUP BY campaign_id, hr
ORDER BY campaign_id, hr`
	rows, err := ch.Query(ctx, query, from, to, campaignIDs)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make(map[uuid.UUID][]VPPCampaignSample, len(campaignIDs))
	for rows.Next() {
		var campID uuid.UUID
		var sample VPPCampaignSample
		if err := rows.Scan(&campID, &sample.HourOfDay, &sample.Impressions); err != nil {
			return nil, err
		}
		out[campID] = append(out[campID], sample)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func hourlySharesFromSamples(samples []VPPCampaignSample) [24]float64 {
	return reports.BuildHourWeights(samples)
}

func ComputeVPPRatio(weights [24]float64, daypart []int16, localNow time.Time, actualSpendMicro, dailyBudgetMicro int64, tolerance float64) float32 {
	if dailyBudgetMicro <= 0 {
		return 1.0
	}
	expectedFrac := SmartPacingExpectedRatio(weights, daypart, localNow)
	expectedSpend := int64(float64(dailyBudgetMicro) * expectedFrac)
	if expectedSpend <= 0 {
		return 1.0
	}
	overThreshold := int64(float64(expectedSpend) * (1.0 + tolerance))
	if actualSpendMicro <= overThreshold {
		return 1.0
	}
	ratio := float64(expectedSpend) / float64(actualSpendMicro)
	if ratio > 1.0 {
		ratio = 1.0
	}
	if ratio < 0 {
		ratio = 0
	}
	return float32(ratio)
}

func VPPPacingRedisKey(campaignID uuid.UUID) string {
	return vppPacingRedisKey(campaignID)
}
