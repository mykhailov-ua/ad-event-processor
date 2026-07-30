package management

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const vppLookbackDays = 7

func (s *Service) queryCampaignVPPSamples(ctx context.Context, campaignID uuid.UUID, from, to time.Time) ([]forecastHourlySample, error) {
	if s.chQuery == nil {
		return nil, nil
	}
	query := `
SELECT toHour(hour) AS hr, sum(impression_count) AS impressions
FROM mv_campaign_hourly_impressions
WHERE hour >= ? AND hour < ? AND campaign_id = ?
GROUP BY hr
ORDER BY hr`
	rows, err := s.chQuery.Query(ctx, query, from, to, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	samples := make([]forecastHourlySample, 0, 24)
	for rows.Next() {
		var sample forecastHourlySample
		if err := rows.Scan(&sample.hourOfDay, &sample.impressions); err != nil {
			return nil, err
		}
		samples = append(samples, sample)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return samples, nil
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
	expectedFrac := smartPacingExpectedRatio(weights, daypart, localNow)
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
