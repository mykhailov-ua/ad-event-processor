package filter

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

type timezoneAttestationMockRepo struct {
	MockRepo
}

func (m *timezoneAttestationMockRepo) ListActiveCampaigns(ctx context.Context) ([]db.ListActiveCampaignsRow, error) {
	return []db.ListActiveCampaignsRow{{
		ID:                      m.ids[0],
		CustomerID:              m.ids[0],
		Status:                  db.CampaignStatusTypeACTIVE,
		Timezone:                "Europe/Warsaw",
		TimezoneAttestationMode: string(domain.TimezoneAttestationModeCampaignTarget),
	}}, nil
}

func TestRegistry_TimezoneAttestationMode_replicaRoundtrip(t *testing.T) {
	id := uuid.New()
	mock := &timezoneAttestationMockRepo{
		MockRepo: MockRepo{
			ids: []pgtype.UUID{{Bytes: id, Valid: true}},
		},
	}

	r := newTestRegistry(t, mock)
	count, err := r.Sync(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, count)

	camp, ok := r.GetCampaign(id)
	require.True(t, ok)
	require.Equal(t, domain.TimezoneAttestationModeCampaignTarget, camp.TimezoneAttestationMode)
}
