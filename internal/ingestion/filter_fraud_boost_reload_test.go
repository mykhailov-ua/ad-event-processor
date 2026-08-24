package ingestion

import (
	"context"
	"sync"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

func TestFilterFraudBoost_ConcurrentReload(t *testing.T) {
	cfg := &config.Config{}
	sw := NewSettingsWatcher(nil, cfg)
	campID := uuid.New()

	engine := NewFilterEngine(0, &fraudSignalsFilter{first: FraudReasonMissingImpTS})
	engine.SetRegistry(&mockRegistry{})
	engine.SetSettingsWatcher(sw)

	cachedMockCamp.Store(&domain.Campaign{ID: campID})
	t.Cleanup(func() { cachedMockCamp.Store(nil) })

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 500 {
			select {
			case <-stop:
				return
			default:
				sw.fraudScoreBoosts.Store(&FraudScoreBoostSnapshot{
					Boosts: map[uuid.UUID]uint8{campID: uint8(i % 50)},
				})
			}
		}
	}()

	wg.Add(8)
	for range 8 {
		go func() {
			defer wg.Done()
			evt := &domain.Event{
				CampaignID:   campID,
				StringBuffer: make([]byte, 0, 64),
			}
			ctx := context.Background()
			for range 200 {
				resetFraudBenchEvent(evt)
				_ = engine.Check(ctx, evt)
			}
		}()
	}
	wg.Wait()
	close(stop)
}
