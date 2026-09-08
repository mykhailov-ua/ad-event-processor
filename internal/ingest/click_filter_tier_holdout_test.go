package ingest

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type debitProbeFilter struct {
	checkCalls int
}

func (f *debitProbeFilter) Check(ctx context.Context, evt *domain.Event) error {
	f.checkCalls++
	return nil
}

func (f *debitProbeFilter) StreamDeferredToProducer() bool { return true }

func (f *debitProbeFilter) SetDeferStreamToProducer(deferWrite bool) {}

func (f *debitProbeFilter) ClickAmountMicro() int64 { return 100 }

func (f *debitProbeFilter) ImpressionAmountMicro() int64 { return 1 }

func (f *debitProbeFilter) LocalQuantaFullSkipEligible(evt *domain.Event, camp *domain.Campaign) bool {
	return false
}

func (f *debitProbeFilter) RollbackDebit(ctx context.Context, evt *domain.Event, camp *domain.Campaign, debitAmount int64, isLocalQuanta bool) {
}

func (f *debitProbeFilter) FinalizeLocalQuantaPublish(ctx context.Context, evt *domain.Event, camp *domain.Campaign) error {
	return nil
}

func (f *debitProbeFilter) SetSkipBudgetDebit(skip bool) {}

func TestFilterEngine_redirectOnlySkipsUnifiedBudgetDebit_holdout(t *testing.T) {
	probe := &debitProbeFilter{}
	engine := NewFilterEngine(time.Second, probe)
	evt := &domain.Event{
		CampaignID:      uuid.New(),
		Type:            "click",
		ClickFilterTier: domain.ClickFilterTierRedirectOnly,
	}
	require.NoError(t, engine.Check(context.Background(), evt))
	require.Equal(t, 0, probe.checkCalls, "redirect_only must not invoke unified budget debit filter")
}

func TestFilterEngine_lightSkipsUnifiedBudgetDebit_holdout(t *testing.T) {
	probe := &debitProbeFilter{}
	engine := NewFilterEngine(time.Second,
		&traceFilter{name: "geo", trace: new([]string)},
		probe,
	)
	evt := &domain.Event{
		CampaignID:      uuid.New(),
		Type:            "click",
		ClickFilterTier: domain.ClickFilterTierLight,
	}
	require.NoError(t, engine.Check(context.Background(), evt))
	require.Equal(t, 0, probe.checkCalls, "light tier must not invoke unified budget debit filter")
}
