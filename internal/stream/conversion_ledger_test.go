package stream

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type stubConversionLedgerStore struct {
	rows map[string]db.ClickConversionLedger
}

func (s *stubConversionLedgerStore) UpsertClickConversionLedger(
	_ context.Context,
	arg db.UpsertClickConversionLedgerParams,
) (db.ClickConversionLedger, error) {
	key := uuid.UUID(arg.CampaignID.Bytes).String() + ":" + arg.ClickID
	prev := s.rows[key]
	total := arg.PayoutMicro
	if prev.ClickID != "" {
		total = prev.PayoutMicro + arg.PayoutMicro
	}
	row := db.ClickConversionLedger{
		CampaignID:  arg.CampaignID,
		ClickID:     arg.ClickID,
		PayoutMicro: total,
		LastStatus:  arg.LastStatus,
	}
	s.rows[key] = row
	return row, nil
}

func TestConversionLedgerApplier_accumulatesSameClickID_holdout(t *testing.T) {
	t.Parallel()
	campID := uuid.New()
	store := &stubConversionLedgerStore{rows: map[string]db.ClickConversionLedger{}}
	applier := NewConversionLedgerApplier(store)

	first := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-accum-1",
		Payload:    []byte(`{"status":"hold","revenue_micro":"1000000"}`),
	}
	second := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-accum-1",
		Payload:    []byte(`{"status":"approved","revenue_micro":"2500000"}`),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{first, second})

	require.Contains(t, string(first.Payload), `"conversion_payout_micro":"1000000"`)
	require.Contains(t, string(second.Payload), `"conversion_payout_micro":"3500000"`)
	require.Contains(t, string(second.Payload), `"payout_micro":"3500000"`)
}

func TestConversionLedgerApplier_skipsWithoutClickID_holdout(t *testing.T) {
	t.Parallel()
	applier := NewConversionLedgerApplier(&stubConversionLedgerStore{rows: map[string]db.ClickConversionLedger{}})
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: uuid.New(),
		Payload:    []byte(`{"revenue_micro":"1000000"}`),
	}
	before := string(evt.Payload)
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	require.Equal(t, before, string(evt.Payload))
}
