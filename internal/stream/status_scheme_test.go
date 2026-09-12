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

type stubStatusSchemeStore struct {
	rows []db.CampaignStatusSchemeRule
}

func (s *stubStatusSchemeStore) ListStatusSchemeRulesByCampaignIDs(
	_ context.Context,
	ids []pgtype.UUID,
) ([]db.CampaignStatusSchemeRule, error) {
	if s == nil || len(s.rows) == 0 {
		return nil, nil
	}
	out := make([]db.CampaignStatusSchemeRule, 0, len(s.rows))
	for _, row := range s.rows {
		for _, id := range ids {
			if row.CampaignID.Valid && id.Bytes == row.CampaignID.Bytes {
				out = append(out, row)
			}
		}
	}
	return out, nil
}

func statusSchemeRule(campID uuid.UUID, whenStatus, internalStatus string, fireOutbound bool) db.CampaignStatusSchemeRule {
	return db.CampaignStatusSchemeRule{
		CampaignID:        pgtype.UUID{Bytes: campID, Valid: true},
		SortOrder:         1,
		WhenStatus:        whenStatus,
		SetInternalStatus: internalStatus,
		PayoutMode:        "inherit",
		FireOutbound:      fireOutbound,
		Enabled:           true,
	}
}

func TestStatusScheme_holdMapsRejected_holdout(t *testing.T) {
	campID := uuid.New()
	applier := NewStatusSchemeApplier(&stubStatusSchemeStore{
		rows: []db.CampaignStatusSchemeRule{
			statusSchemeRule(campID, "hold", "rejected", false),
		},
	})
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		Payload:    []byte(`{"status":"hold","goal_name":"hold"}`),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	require.Contains(t, string(evt.Payload), `"internal_status":"rejected"`)
	require.Contains(t, string(evt.Payload), `"status_scheme_skip_outbound":"true"`)
}

func TestStatusScheme_holdoutNonMatchingStatusPassThrough(t *testing.T) {
	campID := uuid.New()
	applier := NewStatusSchemeApplier(&stubStatusSchemeStore{
		rows: []db.CampaignStatusSchemeRule{
			statusSchemeRule(campID, "hold", "rejected", false),
		},
	})
	original := `{"status":"sale","goal_name":"approved"}`
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		Payload:    []byte(original),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	require.Equal(t, original, string(evt.Payload))
}

func TestStatusScheme_accumulatePayoutMode_holdout(t *testing.T) {
	campID := uuid.New()
	applier := NewStatusSchemeApplier(&stubStatusSchemeStore{
		rows: []db.CampaignStatusSchemeRule{{
			CampaignID:   pgtype.UUID{Bytes: campID, Valid: true},
			SortOrder:    1,
			WhenStatus:   "approved",
			PayoutMode:   "accumulate_payout",
			FireOutbound: true,
			Enabled:      true,
		}},
	})
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		Payload:    []byte(`{"status":"approved","conversion_payout_micro":"4200000"}`),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	require.Contains(t, string(evt.Payload), `"revenue_micro":"4200000"`)
	require.Contains(t, string(evt.Payload), `"payout_micro":"4200000"`)
}

func TestStatusScheme_fixedPayoutMode(t *testing.T) {
	campID := uuid.New()
	applier := NewStatusSchemeApplier(&stubStatusSchemeStore{
		rows: []db.CampaignStatusSchemeRule{{
			CampaignID:   pgtype.UUID{Bytes: campID, Valid: true},
			SortOrder:    1,
			WhenStatus:   "sale",
			PayoutMode:   "fixed",
			PayoutMicro:  2_500_000,
			FireOutbound: true,
			Enabled:      true,
		}},
	})
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		Payload:    []byte(`{"status":"sale"}`),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	require.Contains(t, string(evt.Payload), `"revenue_micro":"2500000"`)
}

func TestStatusSchemeSkipsOutbound(t *testing.T) {
	require.True(t, StatusSchemeSkipsOutbound([]byte(`{"status_scheme_skip_outbound":"true"}`)))
	require.False(t, StatusSchemeSkipsOutbound([]byte(`{"status":"sale"}`)))
}
