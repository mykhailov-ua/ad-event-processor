package runtime

import (
	"testing"
	"time"

	"ad-event-processor/internal/campaign"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSynthesizeHourlyBuckets_preservesTotals(t *testing.T) {
	campaignID := uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	end := mustParseRFC3339(t, "2026-08-31T18:00:00Z")

	buckets := synthesizeHourlyBuckets(campaignID, 12_483, 907, 43, end)
	require.Len(t, buckets, 24)

	var imp, clk, conv int64
	for _, bucket := range buckets {
		imp += bucket.Impressions
		clk += bucket.Clicks
		conv += bucket.Conversions
	}
	require.Equal(t, int64(12_483), imp)
	require.Equal(t, int64(907), clk)
	require.Equal(t, int64(43), conv)
	require.Equal(t, "2026-08-30T19:00:00Z", buckets[0].Hour)
	require.Equal(t, "2026-08-31T18:00:00Z", buckets[23].Hour)
}

func TestSynthesizeHourlyBuckets_zeroTotals_returnsLabeledBuckets(t *testing.T) {
	campaignID := uuid.MustParse("00000000-0000-0000-0000-00000000000b")
	end := mustParseRFC3339(t, "2026-08-31T12:00:00Z")

	buckets := synthesizeHourlyBuckets(campaignID, 0, 0, 0, end)
	require.Len(t, buckets, 24)
	require.Equal(t, "2026-08-31T12:00:00Z", buckets[23].Hour)
}

func TestHourlyBucketsForReport_fallsBackToPGWhenCHEmpty(t *testing.T) {
	campaignID := uuid.MustParse("00000000-0000-0000-0000-00000000000c")
	end := mustParseRFC3339(t, "2026-08-31T18:00:00Z")

	resolved := hourlyBucketsForReport(campaignID, 12_483, 907, 43, end, nil)
	require.Len(t, resolved, 24)

	var imp, clk int64
	for _, bucket := range resolved {
		imp += bucket.Impressions
		clk += bucket.Clicks
	}
	require.Equal(t, int64(12_483), imp)
	require.Equal(t, int64(907), clk)
}

func TestHourlyBucketsForReport_prefersCHWhenActive(t *testing.T) {
	campaignID := uuid.MustParse("00000000-0000-0000-0000-00000000000d")
	end := mustParseRFC3339(t, "2026-08-31T18:00:00Z")
	chHourly := []campaign.CampaignHourlyBucketDTO{
		{Hour: end.Format(time.RFC3339), Impressions: 42, Clicks: 7, Conversions: 1},
	}

	resolved := hourlyBucketsForReport(campaignID, 99_999, 88_888, 77, end, chHourly)
	require.Equal(t, chHourly, resolved)
}

func TestPgHourlyWeights_holdoutNotUniform(t *testing.T) {
	idA := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	idB := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	wA := pgHourlyWeights(idA)
	wB := pgHourlyWeights(idB)

	var sumA, sumB float64
	for i := range wA {
		sumA += wA[i]
		sumB += wB[i]
	}
	require.InDelta(t, 1.0, sumA, 0.0001)
	require.InDelta(t, 1.0, sumB, 0.0001)
	require.NotEqual(t, wA, wB)
}

func mustParseRFC3339(t *testing.T, raw string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, raw)
	require.NoError(t, err)
	return parsed.UTC()
}
