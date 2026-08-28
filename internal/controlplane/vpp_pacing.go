package controlplane

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/campaign"

	"github.com/google/uuid"
)

const (
	vppLookbackDays        = 7
	vppCampaignSampleBatch = 200
)

func (s *Service) queryVPPCampaignSamplesBatch(ctx context.Context, from, to time.Time, campaignIDs []uuid.UUID) (map[uuid.UUID][]forecastHourlySample, error) {
	if s.clickhouseQuery == nil || len(campaignIDs) == 0 {
		return map[uuid.UUID][]forecastHourlySample{}, nil
	}
	clickhouseCtx, cancel := clickhouseQueryContext(ctx)
	defer cancel()

	out := make(map[uuid.UUID][]forecastHourlySample, len(campaignIDs))
	for start := 0; start < len(campaignIDs); start += vppCampaignSampleBatch {
		end := start + vppCampaignSampleBatch
		if end > len(campaignIDs) {
			end = len(campaignIDs)
		}
		chunk, err := s.queryVPPCampaignSamplesChunk(clickhouseCtx, from, to, campaignIDs[start:end])
		if err != nil {
			return nil, err
		}
		for id, samples := range chunk {
			out[id] = samples
		}
	}
	return out, nil
}

func (s *Service) queryVPPCampaignSamplesChunk(ctx context.Context, from, to time.Time, campaignIDs []uuid.UUID) (map[uuid.UUID][]forecastHourlySample, error) {
	query := `
SELECT campaign_id, toHour(hour) AS hr, sum(impression_count) AS impressions
FROM mv_campaign_hourly_impressions
WHERE hour >= ? AND hour < ? AND campaign_id IN (?)
GROUP BY campaign_id, hr
ORDER BY campaign_id, hr`
	rows, err := s.clickhouseQuery.Query(ctx, query, from, to, campaignIDs)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make(map[uuid.UUID][]forecastHourlySample, len(campaignIDs))
	for rows.Next() {
		var campID uuid.UUID
		var sample forecastHourlySample
		if err := rows.Scan(&campID, &sample.hourOfDay, &sample.impressions); err != nil {
			return nil, err
		}
		out[campID] = append(out[campID], sample)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func hourlySharesFromSamples(samples []forecastHourlySample) [24]float64 {
	return buildHourWeights(samples)
}

func sumHourlyShares(weights [24]float64) float64 {
	var sum float64
	for _, w := range weights {
		sum += w
	}
	return sum
}

func computeVPPRatio(weights [24]float64, daypart []int16, localNow time.Time, actualSpendMicro, dailyBudgetMicro int64, tolerance float64) float32 {
	if dailyBudgetMicro <= 0 {
		return 1.0
	}
	expectedFrac := campaign.SmartPacingExpectedRatio(weights, daypart, localNow)
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

func vppPacingRedisKey(campaignID uuid.UUID) string {
	return fmt.Sprintf("campaign:%s:pacing", campaignID.String())
}
