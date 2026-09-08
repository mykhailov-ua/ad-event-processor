package stream

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

func TestConversionPayoutApplier_mapsSaleStatusFromPreset_holdout(t *testing.T) {
	campID := uuid.New()
	store := &stubConversionMappingStore{
		rows: []db.CampaignConversionMapping{{
			CampaignID:    pgtype.UUID{Bytes: campID, Valid: true},
			InboundStatus: "sale",
			GoalName:      "approved",
			PayoutMicro:   3_500_000,
		}},
	}
	applier := NewConversionPayoutApplier(store)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		Payload:    []byte(`{"status":"sale"}`),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	require.Contains(t, string(evt.Payload), `"goal_name":"approved"`)
	require.Contains(t, string(evt.Payload), "3500000")
}

func TestConversionPayoutApplier_leadStatusAlias(t *testing.T) {
	campID := uuid.New()
	store := &stubConversionMappingStore{
		rows: []db.CampaignConversionMapping{{
			CampaignID:    pgtype.UUID{Bytes: campID, Valid: true},
			InboundStatus: "hold",
			GoalName:      "hold",
			PayoutMicro:   0,
		}},
	}
	applier := NewConversionPayoutApplier(store)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		Payload:    []byte(`{"lead_status":"hold"}`),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	require.Contains(t, string(evt.Payload), `"goal_name":"hold"`)
}

func TestConversionPayoutApplier_unknownStatusPassThrough(t *testing.T) {
	campID := uuid.New()
	store := &stubConversionMappingStore{
		rows: []db.CampaignConversionMapping{{
			CampaignID:    pgtype.UUID{Bytes: campID, Valid: true},
			InboundStatus: "sale",
			GoalName:      "approved",
			PayoutMicro:   1_000_000,
		}},
	}
	applier := NewConversionPayoutApplier(store)
	original := `{"status":"unknown_ext"}`
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		Payload:    []byte(original),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	require.Equal(t, original, string(evt.Payload))
}
