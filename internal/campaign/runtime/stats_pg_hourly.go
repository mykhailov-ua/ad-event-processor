package runtime

import (
	"math"
	"sort"
	"time"

	"ad-event-processor/internal/campaign"

	"github.com/google/uuid"
)

// synthesizeHourlyBuckets spreads PG daily totals across 24 hourly buckets for admin UI
// when ClickHouse is not wired (ingest-only dev). Totals per metric match inputs.
func synthesizeHourlyBuckets(
	campaignID uuid.UUID,
	impressions, clicks, conversions int64,
	end time.Time,
) []campaign.CampaignHourlyBucketDTO {
	weights := pgHourlyWeights(campaignID)
	end = end.UTC().Truncate(time.Hour)

	impParts := distributeInt64(impressions, weights)
	clkParts := distributeInt64(clicks, weights)
	convParts := distributeInt64(conversions, weights)

	buckets := make([]campaign.CampaignHourlyBucketDTO, 24)
	for i := range buckets {
		hourStart := end.Add(-time.Duration(23-i) * time.Hour)
		buckets[i] = campaign.CampaignHourlyBucketDTO{
			Hour:        hourStart.Format(time.RFC3339),
			Impressions: impParts[i],
			Clicks:      clkParts[i],
			Conversions: convParts[i],
		}
	}
	return buckets
}

func pgHourlyWeights(campaignID uuid.UUID) [24]float64 {
	var weights [24]float64
	seed := int(campaignID[0]) + int(campaignID[7])*3 + int(campaignID[15])*11
	var sum float64
	for h := 0; h < 24; h++ {
		peak := 0.45 + 0.55*math.Sin(float64(h-10)*math.Pi/12.0)
		if peak < 0.12 {
			peak = 0.12
		}
		noise := float64((seed+h*19)%37) / 100.0
		weights[h] = peak + noise
		sum += weights[h]
	}
	if sum <= 0 {
		weights[0] = 1
		return weights
	}
	for h := 0; h < 24; h++ {
		weights[h] /= sum
	}
	return weights
}

func distributeInt64(total int64, weights [24]float64) [24]int64 {
	var out [24]int64
	if total <= 0 {
		return out
	}

	type fracSlot struct {
		idx int
		rem float64
	}
	fracs := make([]fracSlot, 0, 24)
	var assigned int64
	for i, w := range weights {
		raw := float64(total) * w
		base := int64(math.Floor(raw))
		out[i] = base
		assigned += base
		fracs = append(fracs, fracSlot{idx: i, rem: raw - float64(base)})
	}

	remainder := total - assigned
	sort.Slice(fracs, func(a, b int) bool {
		return fracs[a].rem > fracs[b].rem
	})
	for i := int64(0); i < remainder; i++ {
		out[fracs[i%24].idx]++
	}
	return out
}

func hourlyBucketsHaveActivity(buckets []campaign.CampaignHourlyBucketDTO) bool {
	for _, bucket := range buckets {
		if bucket.Impressions > 0 || bucket.Clicks > 0 || bucket.Conversions > 0 {
			return true
		}
	}
	return false
}

// hourlyBucketsForReport prefers ClickHouse hourly buckets when they carry activity.
// When CH is wired but empty (ingest-only dev, lag, or new campaign), synthesize from PG
// daily totals so admin charts still render.
func hourlyBucketsForReport(
	campaignID uuid.UUID,
	impressions, clicks, conversions int64,
	to time.Time,
	chHourly []campaign.CampaignHourlyBucketDTO,
) []campaign.CampaignHourlyBucketDTO {
	if hourlyBucketsHaveActivity(chHourly) {
		return chHourly
	}
	if impressions == 0 && clicks == 0 && conversions == 0 {
		return chHourly
	}
	return synthesizeHourlyBuckets(campaignID, impressions, clicks, conversions, to)
}
