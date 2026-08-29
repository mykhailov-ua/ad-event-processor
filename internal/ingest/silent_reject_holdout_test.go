package ingest

import (
	"context"
	"net/http"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFraudReject_holdoutSilentRejectFlag(t *testing.T) {
	campID := uuid.New()
	reg := stubCampaignRegistry{
		ok: true,
		camp: &domain.Campaign{
			ID: campID,
		},
	}
	engine := NewFilterEngine(0, &errFilter{err: ErrFraudDetected})
	engine.SetRegistry(reg)
	proc := newTrackProcessor(engine, reg, nil)

	t.Run("holdoutPositive_silentRejectEnabled", func(t *testing.T) {
		reg.camp.SilentRejectEnabled = true
		evt := domain.EventPool.Get().(*domain.Event)
		evt.Reset()
		defer domain.EventPool.Put(evt)
		evt.CampaignID = campID

		out := processTrack(context.Background(), proc, evt, nil)
		require.Equal(t, trackStatusFraudAccepted, out.Status)
		require.Equal(t, filterRejectFraud, out.RejectKind)
		assert.True(t, evt.SilentRejectEvent)
		assert.Equal(t, http.StatusAccepted, filterRejectSpecs[out.RejectKind].status)
	})

	t.Run("holdoutNegative_silentRejectDisabled", func(t *testing.T) {
		reg.camp.SilentRejectEnabled = false
		evt := domain.EventPool.Get().(*domain.Event)
		evt.Reset()
		defer domain.EventPool.Put(evt)
		evt.CampaignID = campID

		out := processTrack(context.Background(), proc, evt, nil)
		require.Equal(t, trackStatusRejected, out.Status)
		require.Equal(t, filterRejectFraudBlocked, out.RejectKind)
		assert.False(t, evt.SilentRejectEvent)
		assert.Equal(t, http.StatusForbidden, filterRejectSpecs[out.RejectKind].status)
	})
}
