package filter

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type antifraudCampRegistry struct {
	camp *domain.Campaign
}

func (r *antifraudCampRegistry) Exists(_ uuid.UUID) bool { return r.camp != nil }
func (r *antifraudCampRegistry) Add(_ uuid.UUID, _ uuid.UUID, _ *uuid.UUID, _ string, _ domain.PacingMode, _ int64, _ string, _ int32, _ int32, _ []string) {
}
func (r *antifraudCampRegistry) GetCustomerID(_ uuid.UUID) (uuid.UUID, bool) { return uuid.Nil, false }
func (r *antifraudCampRegistry) GetCampaign(_ uuid.UUID) (*domain.Campaign, bool) {
	if r.camp == nil {
		return nil, false
	}
	return r.camp, true
}
func (r *antifraudCampRegistry) Sync(_ context.Context) (int, error)          { return 0, nil }
func (r *antifraudCampRegistry) StartSync(_ context.Context, _ time.Duration) {}
func (r *antifraudCampRegistry) Wait(_ context.Context) error                 { return nil }

func TestAntifraudTelemetryFilter_automation_holdout(t *testing.T) {
	campID := uuid.MustParse("00000000-0000-4000-8000-000000000099")
	reg := &antifraudCampRegistry{
		camp: &domain.Campaign{
			ID:                 campID,
			SafePageEnabled:    true,
			AttestationEnabled: true,
		},
	}
	f := NewAntifraudTelemetryFilter(reg)
	f.SetEnabled(true)
	evt := &domain.Event{
		Type:         "conversion",
		CampaignID:   campID,
		UA:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		AntifraudSet: 1,
		AntifraudSnapshot: domain.AntifraudSnapshot{
			Webdriver: 1,
		},
	}
	acc := AttachFraudAccumulator(evt)
	require.NoError(t, f.Check(context.Background(), evt))
	found := false
	for i := 0; i < int(acc.count); i++ {
		if acc.signals[i] == FraudReasonAntifraudAutomationLeak {
			found = true
			break
		}
	}
	require.True(t, found)
}
