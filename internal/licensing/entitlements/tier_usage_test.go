package entitlements

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTierUsageWarnings_campaignCap(t *testing.T) {
	limits := Limits{MaxActiveCampaigns: 10}
	w := TierUsageWarnings(limits, 10, StateActive, time.Time{}, time.Now(), 7)
	assert.Contains(t, w[0], "cap reached")

	w = TierUsageWarnings(limits, 8, StateActive, time.Time{}, time.Now(), 7)
	assert.Contains(t, w[0], "Approaching")
}

func TestTierUsageWarnings_renewalWindow(t *testing.T) {
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	validUntil := now.Add(5 * 24 * time.Hour)
	w := TierUsageWarnings(Limits{}, 0, StateActive, validUntil, now, 7)
	assert.Len(t, w, 1)
	assert.Contains(t, w[0], "renews in 5 day")
}

func TestTierUsageWarnings_graceState(t *testing.T) {
	w := TierUsageWarnings(Limits{}, 0, StateGrace, time.Time{}, time.Now(), 7)
	assert.Contains(t, w[0], "grace period")
}
