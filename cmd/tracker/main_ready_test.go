package main_test

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingestion"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCampaignRegistry struct{}

func (stubCampaignRegistry) Exists(uuid.UUID) bool { return true }
func (stubCampaignRegistry) Add(uuid.UUID, uuid.UUID, *uuid.UUID, string, domain.PacingMode, int64, string, int32, int32, []string) {
}
func (stubCampaignRegistry) GetCustomerID(uuid.UUID) (uuid.UUID, bool) { return uuid.Nil, true }
func (stubCampaignRegistry) GetCampaign(uuid.UUID) (*domain.Campaign, bool) {
	return nil, false
}
func (stubCampaignRegistry) Sync(context.Context) (int, error)        { return 0, nil }
func (stubCampaignRegistry) StartSync(context.Context, time.Duration) {}
func (stubCampaignRegistry) Wait(context.Context) error               { return nil }

func newReadyTestHandler(t *testing.T, warmupSec int) *ingestion.AdsPacketHandler {
	t.Helper()
	cfg := &config.Config{NodeWarmupSec: warmupSec}
	return ingestion.NewAdsPacketHandler(cfg, stubCampaignRegistry{}, nil, nil, nil, ingestion.NewJumpHashSharder(1), "fraud", nil)
}

func TestTrackerHealth_LivenessOnly(t *testing.T) {
	h := newReadyTestHandler(t, 300)
	h.SetHealthProbeState(false)

	status, body := ingestion.GetHealthGnet(h)
	require.Equal(t, 200, status)
	assert.Contains(t, body, "OK")
}

func TestTrackerReady_BeforeWarmup_503(t *testing.T) {
	h := newReadyTestHandler(t, 300)
	h.SetHealthProbeState(true)

	status, body := ingestion.GetReadyGnet(h)
	require.Equal(t, 503, status)
	assert.Contains(t, body, "not ready")
}

func TestTrackerReady_AfterWarmup_200(t *testing.T) {
	h := newReadyTestHandler(t, 300)
	h.SetHealthProbeState(true)
	h.SetStartedAt(time.Now().Add(-301 * time.Second))

	status, body := ingestion.GetReadyGnet(h)
	require.Equal(t, 200, status)
	assert.Contains(t, body, "OK")
}
