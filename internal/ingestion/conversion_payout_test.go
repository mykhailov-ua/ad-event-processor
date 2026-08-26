package ingestion

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

type stubConversionMappingStore struct {
	rows []db.CampaignConversionMapping
}

func (s *stubConversionMappingStore) ListConversionMappingsByCampaignIDs(
	_ context.Context,
	ids []pgtype.UUID,
) ([]db.CampaignConversionMapping, error) {
	if s == nil || len(s.rows) == 0 {
		return nil, nil
	}
	out := make([]db.CampaignConversionMapping, 0, len(s.rows))
	for _, row := range s.rows {
		for _, id := range ids {
			if row.CampaignID.Valid && id.Bytes == row.CampaignID.Bytes {
				out = append(out, row)
			}
		}
	}
	return out, nil
}

func TestConversionPayoutApplier_skipsValidationPending(t *testing.T) {
	campID := uuid.New()
	store := &stubConversionMappingStore{
		rows: []db.CampaignConversionMapping{{
			CampaignID:    pgtype.UUID{Bytes: campID, Valid: true},
			InboundStatus: "approved",
			GoalName:      "lead",
			PayoutMicro:   2_500_000,
		}},
	}
	applier := NewConversionPayoutApplier(store)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		Payload:    []byte(`{"status":"approved","conversion_validation_pending":true,"revenue_micro":0}`),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	fields := parsePayloadStringFields(evt.Payload)
	if fields["goal_name"] == "lead" && fields["revenue_micro"] == "2500000" {
		t.Fatalf("payout applied on pending: %s", evt.Payload)
	}
	require.NotContains(t, string(evt.Payload), "2500000")
}
