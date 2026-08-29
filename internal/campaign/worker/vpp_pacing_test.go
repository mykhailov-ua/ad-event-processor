package worker

import (
	"testing"
	"time"

	"ad-event-processor/internal/campaign"

	"github.com/stretchr/testify/require"
)

func TestHourlySharesFromSamples_sumsToOne(t *testing.T) {
	samples := []VPPCampaignSample{
		{HourOfDay: 8, Impressions: 100},
		{HourOfDay: 9, Impressions: 300},
		{HourOfDay: 10, Impressions: 100},
	}
	weights := hourlySharesFromSamples(samples)
	var sum float64
	for _, w := range weights {
		sum += w
	}
	require.InDelta(t, 1.0, sum, 1e-9)
}

func TestComputeVPPRatio_flatDistributionNearOne(t *testing.T) {
	weights := campaign.UniformHourWeights()
	now := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	ratio := ComputeVPPRatio(weights, nil, now, 50_000_000, 100_000_000, 0.05)
	require.Equal(t, float32(1.0), ratio)
}

func TestComputeVPPRatio_spikeMorningLowersRatio(t *testing.T) {
	var weights [24]float64
	for h := 6; h <= 10; h++ {
		weights[h] = 0.2
	}
	morning := time.Date(2026, 3, 15, 9, 30, 0, 0, time.UTC)
	ratio := ComputeVPPRatio(weights, nil, morning, 80_000_000, 100_000_000, 0.0)
	require.Less(t, ratio, float32(1.0))
	require.Greater(t, ratio, float32(0.0))
}
