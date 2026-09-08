package filter

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

type countingStaleRepo struct {
	MockRepo
	calls atomic.Int32
}

func (r *countingStaleRepo) GetCampaignFull(ctx context.Context, id pgtype.UUID) (db.GetCampaignFullRow, error) {
	r.calls.Add(1)
	return r.MockRepo.GetCampaignFull(ctx, id)
}

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

func TestLookupCampaign_stalePGGraceDisabled_noPGQueries_holdout(t *testing.T) {
	campID := uuid.New()
	repo := &countingStaleRepo{MockRepo: MockRepo{}}
	reg := staleRegistry(t, repo)
	reg.SetStalePGGrace(false)

	evt := &domain.Event{CampaignID: campID, IP: "8.8.8.8"}
	_, err := LookupCampaign(context.Background(), reg, evt)
	require.ErrorIs(t, err, ErrRegistryStale)
	require.Equal(t, int32(0), repo.calls.Load())
}

func TestLookupCampaign_stalePGGrace_circuitOpen503_holdout(t *testing.T) {
	campID1 := uuid.New()
	campID2 := uuid.New()
	custID := uuid.New()
	repo := &MockRepo{
		full: map[uuid.UUID]db.GetCampaignFullRow{
			campID1: {
				ID:         pgtype.UUID{Bytes: campID1, Valid: true},
				CustomerID: pgtype.UUID{Bytes: custID, Valid: true},
				Status:     db.CampaignStatusTypeACTIVE,
			},
			campID2: {
				ID:         pgtype.UUID{Bytes: campID2, Valid: true},
				CustomerID: pgtype.UUID{Bytes: custID, Valid: true},
				Status:     db.CampaignStatusTypeACTIVE,
			},
		},
	}
	reg := staleRegistry(t, repo)
	reg.SetStalePGMaxRPS(1)

	evt1 := &domain.Event{CampaignID: campID1, IP: "8.8.8.8"}
	_, err := LookupCampaign(context.Background(), reg, evt1)
	require.NoError(t, err)

	evt2 := &domain.Event{CampaignID: campID2, IP: "8.8.8.8"}
	_, err = LookupCampaign(context.Background(), reg, evt2)
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
