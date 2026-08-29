package ingest

import (
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLocalTTCCache_lowTTC(t *testing.T) {
	cache := NewLocalTTCCache()
	campID := uuid.New()
	cache.Record(campID, "user-a")
	outcome := cache.CheckClick(campID, "user-a", 5000, false)
	assert.Equal(t, localTTCLow, outcome)
}

func TestLocalTTCCache_missingImpBypass(t *testing.T) {
	cache := NewLocalTTCCache()
	campID := uuid.New()
	outcome := cache.CheckClick(campID, "unknown", 100, false)
	assert.Equal(t, localTTCBypass, outcome)
}

func TestLocalTTCCache_missingImpFailClosed(t *testing.T) {
	cache := NewLocalTTCCache()
	campID := uuid.New()
	outcome := cache.CheckClick(campID, "unknown", 100, true)
	assert.Equal(t, localTTCMissingClosed, outcome)
}

func TestUnifiedFilter_applyGoTTC_attestationFailClosed(t *testing.T) {
	campID := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = campID
		c.AttestationMode = domain.AttestationModeLight
		c.AttestationEnabled = false
	})
	cachedMockCamp.Store(nil)

	f := NewUnifiedFilter(nil, NewJumpHashSharder(1), &mockRegistry{}, nil, 0, time.Minute, time.Hour, time.Hour, 100, 10, "events", 1000)
	f.SetTTCMin(time.Second)
	f.SetLocalTTCCache(NewLocalTTCCache())

	evt := &domain.Event{
		Type:       "click",
		CampaignID: campID,
		UserID:     "no-imp-user",
	}
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	f.applyGoTTC(evt)
	assert.True(t, acc.has(FraudReasonMissingImpTS))
}

func TestUnifiedFilter_applyGoTTC_signalsLowTTC(t *testing.T) {
	f := NewUnifiedFilter(nil, nil, &mockRegistry{}, nil, 0, time.Minute, time.Hour, time.Hour, 100, 10, "events", 1000)
	f.SetTTCMin(time.Second)
	f.SetLocalTTCCache(NewLocalTTCCache())

	campID := uuid.New()
	f.localTTC.Record(campID, "ttc-user")

	evt := &domain.Event{
		Type:       "click",
		CampaignID: campID,
		UserID:     "ttc-user",
	}
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	f.applyGoTTC(evt)
	assert.True(t, acc.has(FraudReasonLowTTC))
}

func TestRoughPacingGate_rejectsBurst(t *testing.T) {
	gate := NewRoughPacingGate()
	campID := uuid.New()
	const daily = int64(24_000_000)
	assert.True(t, gate.Allow(campID, 500_000, daily, 1))
	assert.False(t, gate.Allow(campID, 600_000, daily, 1))
}
