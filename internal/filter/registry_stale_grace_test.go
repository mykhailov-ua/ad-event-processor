package filter

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

type staleGraceRepo struct {
	MockRepo
	missing map[uuid.UUID]struct{}
}

func (r *staleGraceRepo) GetCampaignFull(ctx context.Context, id pgtype.UUID) (db.GetCampaignFullRow, error) {
	if r.missing != nil {
		if _, ok := r.missing[uuid.UUID(id.Bytes)]; ok {
			return db.GetCampaignFullRow{}, pgx.ErrNoRows
		}
	}
	return r.MockRepo.GetCampaignFull(ctx, id)
}

func staleRegistry(t *testing.T, repo db.Querier) *Registry {
	t.Helper()
	reg := NewRegistry(repo)
	reg.ConfigureStaleMode(1 * time.Millisecond)
	time.Sleep(3 * time.Millisecond)
	require.True(t, reg.IsStaleMode())
	return reg
}

func TestLookupCampaign_stalePGGrace_warmsKnownCampaign(t *testing.T) {
	campID := uuid.New()
	custID := uuid.New()
	repo := &MockRepo{
		full: map[uuid.UUID]db.GetCampaignFullRow{
			campID: {
				ID:         pgtype.UUID{Bytes: campID, Valid: true},
				CustomerID: pgtype.UUID{Bytes: custID, Valid: true},
				Status:     db.CampaignStatusTypeACTIVE,
			},
		},
	}
	reg := staleRegistry(t, repo)

	evt := &domain.Event{CampaignID: campID, IP: "8.8.8.8"}
	camp, err := LookupCampaign(context.Background(), reg, evt)
	require.NoError(t, err)
	require.Equal(t, campID, camp.ID)
}

func TestLookupCampaign_stalePGGrace_unknownCampaign404(t *testing.T) {
	campID := uuid.New()
	repo := &staleGraceRepo{
		MockRepo: MockRepo{},
		missing:  map[uuid.UUID]struct{}{campID: {}},
	}
	reg := staleRegistry(t, repo)

	evt := &domain.Event{CampaignID: campID, IP: "8.8.8.8"}
	_, err := LookupCampaign(context.Background(), reg, evt)
	require.ErrorIs(t, err, ErrCampaignNotFound)
}

func TestLookupCampaign_stalePGGraceDisabled503_holdout(t *testing.T) {
	campID := uuid.New()
	reg := staleRegistry(t, &MockRepo{})
	reg.SetStalePGGrace(false)

	evt := &domain.Event{CampaignID: campID, IP: "8.8.8.8"}
	_, err := LookupCampaign(context.Background(), reg, evt)
	require.ErrorIs(t, err, ErrRegistryStale)
}

func TestGeoFilter_stalePGGrace_knownCampaignPasses(t *testing.T) {
	campID := uuid.New()
	custID := uuid.New()
	repo := &MockRepo{
		full: map[uuid.UUID]db.GetCampaignFullRow{
			campID: {
				ID:         pgtype.UUID{Bytes: campID, Valid: true},
				CustomerID: pgtype.UUID{Bytes: custID, Valid: true},
				Status:     db.CampaignStatusTypeACTIVE,
			},
		},
	}
	reg := staleRegistry(t, repo)
	f := NewGeoFilter(&MockGeoProvider{}, reg)

	evt := &domain.Event{CampaignID: campID, IP: "8.8.8.8"}
	require.NoError(t, f.Check(context.Background(), evt))
}
