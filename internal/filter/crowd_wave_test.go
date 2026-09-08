package filter

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/crowdwave"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func synthWaveClusterID(i int) [16]byte {
	var id [16]byte
	id[0] = byte(i + 1)
	return id
}

func TestCrowdWaveFilter_holdoutSyntheticWaveTriggers(t *testing.T) {
	campID := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	before := testutil.ToFloat64(metrics.HybridCrowdWaveTotal.WithLabelValues(campID.String()))
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	policy := crowdwave.WavePolicy{
		WindowSec:           3600,
		MinUniqueClusters:   10,
		MinSimhashNeighbors: 8,
		SimhashMaxDist:      2,
	}
	store := NewCrowdWaveStore(client, time.Hour, policy)
	reg := &crowdProbeCampRegistry{
		camp: &domain.Campaign{
			ID:                 campID,
			SafePageEnabled:    true,
			AttestationEnabled: true,
		},
	}
	f := NewCrowdWaveFilter(reg)
	f.SetEnabled(true)
	f.SetStore(store)
	ctx := context.Background()
	var evt *domain.Event
	for i := 0; i < 10; i++ {
		evt = &domain.Event{
			Type:            "click",
			CampaignID:      campID,
			ProbeClusterSet: 1,
			ProbeClusterID:  synthWaveClusterID(i),
			TelemetrySet:    1,
			TelemetryEvents: []domain.BehaviorTelemetryEvent{
				{T: "pointerdown"}, {T: "scroll"}, {T: "click"},
			},
		}
		err := f.Check(ctx, evt)
		require.NoError(t, err)
		if i < 9 {
			assert.Equal(t, uint8(0), evt.CrowdWaveActive)
		}
	}
	assert.Equal(t, uint8(1), evt.CrowdWaveActive)
	assert.Greater(t, evt.CrowdWaveScore, uint16(0))
	assert.GreaterOrEqual(t, testutil.ToFloat64(metrics.HybridCrowdWaveTotal.WithLabelValues(campID.String())), before+1)
}

func TestCrowdWaveFilter_holdoutOrganicSpreadDoesNotTrigger(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewCrowdWaveStore(client, time.Hour, crowdwave.DefaultWavePolicy())
	campID := uuid.New()
	reg := &crowdProbeCampRegistry{
		camp: &domain.Campaign{
			ID:                 campID,
			SafePageEnabled:    true,
			AttestationEnabled: true,
		},
	}
	f := NewCrowdWaveFilter(reg)
	f.SetEnabled(true)
	f.SetStore(store)
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		events := make([]domain.BehaviorTelemetryEvent, i+3)
		for j := range events {
			events[j] = domain.BehaviorTelemetryEvent{T: eventTypeFromIndex(i + j)}
		}
		evt := &domain.Event{
			Type:            "click",
			CampaignID:      campID,
			ProbeClusterSet: 1,
			ProbeClusterID:  synthWaveClusterID(i),
			TelemetrySet:    1,
			TelemetryEvents: events,
		}
		err := f.Check(ctx, evt)
		require.NoError(t, err)
		assert.Equal(t, uint8(0), evt.CrowdWaveActive)
	}
	state, err := store.Snapshot(ctx, campID)
	require.NoError(t, err)
	assert.False(t, state.Active)
}

func eventTypeFromIndex(i int) string {
	switch i % 7 {
	case 0:
		return "pointerdown"
	case 1:
		return "click"
	case 2:
		return "mousemove"
	case 3:
		return "scroll"
	case 4:
		return "keydown"
	case 5:
		return "visibilitychange"
	default:
		return "touchstart"
	}
}
