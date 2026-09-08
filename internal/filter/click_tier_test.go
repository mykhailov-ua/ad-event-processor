package filter

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/licensing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterEngine_lightTierSkipsUnified(t *testing.T) {
	var order []string
	engine := NewFilterEngine(time.Second,
		NewLicenseFilter(&stubLicenseRegistry{state: licensing.StateActive}),
		NewEmergencyBreakerFilter(nil),
		&clickTierTraceFilter{name: "unified", trace: &order},
	)
	evt := &domain.Event{
		CampaignID:      uuid.New(),
		ClickFilterTier: domain.ClickFilterTierLight,
	}
	require.NoError(t, engine.Check(context.Background(), evt))
	assert.Empty(t, order, "light tier must skip unified debit filter")
}

func TestFilterEngine_redirectOnlyRunsLicenseOnly(t *testing.T) {
	var order []string
	engine := NewFilterEngine(time.Second,
		NewLicenseFilter(&stubLicenseRegistry{state: licensing.StateActive}),
		NewEmergencyBreakerFilter(nil),
		&clickTierTraceFilter{name: "unified", trace: &order},
	)
	evt := &domain.Event{
		CampaignID:      uuid.New(),
		ClickFilterTier: domain.ClickFilterTierRedirectOnly,
	}
	require.NoError(t, engine.Check(context.Background(), evt))
	assert.Empty(t, order)
}

func TestClickTierSkipsUnifiedFilter_holdout(t *testing.T) {
	assert.True(t, ClickTierSkipsUnifiedFilter(domain.ClickFilterTierRedirectOnly))
	assert.True(t, ClickTierSkipsUnifiedFilter(domain.ClickFilterTierLight))
	assert.False(t, ClickTierSkipsUnifiedFilter(domain.ClickFilterTierFull))
}

type clickTierTraceFilter struct {
	name  string
	trace *[]string
	fail  error
}

func (f *clickTierTraceFilter) Check(ctx context.Context, evt *domain.Event) error {
	*f.trace = append(*f.trace, f.name)
	return f.fail
}

func TestFilterEngine_redirectOnlyWithLicenseFilter(t *testing.T) {
	engine := NewFilterEngine(0, NewLicenseFilter(&stubLicenseRegistry{state: licensing.StateActive}))
	evt := &domain.Event{
		CampaignID:      uuid.New(),
		ClickFilterTier: domain.ClickFilterTierRedirectOnly,
	}
	require.NoError(t, engine.Check(context.Background(), evt))
}
