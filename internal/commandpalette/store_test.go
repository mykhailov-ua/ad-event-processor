package commandpalette

import (
	"context"
	"os"
	"strings"
	"testing"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingQuerier struct {
	campaignLimit int32
}

func (r *recordingQuerier) SearchCommandPaletteCampaigns(_ context.Context, arg db.SearchCommandPaletteCampaignsParams) ([]db.SearchCommandPaletteCampaignsRow, error) {
	r.campaignLimit = arg.ResultLimit
	return nil, nil
}

func (r *recordingQuerier) SearchCommandPaletteFlows(context.Context, db.SearchCommandPaletteFlowsParams) ([]db.SearchCommandPaletteFlowsRow, error) {
	return nil, nil
}

func (r *recordingQuerier) SearchCommandPaletteLanders(context.Context, db.SearchCommandPaletteLandersParams) ([]db.SearchCommandPaletteLandersRow, error) {
	return nil, nil
}

func (r *recordingQuerier) SearchCommandPaletteOffers(context.Context, db.SearchCommandPaletteOffersParams) ([]db.SearchCommandPaletteOffersRow, error) {
	return nil, nil
}

func TestStore_SearchEntities_passesPerKindLimit(t *testing.T) {
	t.Parallel()
	rec := &recordingQuerier{}
	st := &Store{q: rec}
	customerID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	_, err := st.SearchEntities(t.Context(), customerID, "camp", 25, []string{"campaign"})
	require.NoError(t, err)
	assert.Equal(t, int32(perKindSearchBudget), rec.campaignLimit)
}

func TestCommandPalette_search_noThousandRowScan_holdout(t *testing.T) {
	t.Parallel()
	sqlBytes, err := os.ReadFile("../../internal/ingest/queries/command_palette.sql")
	require.NoError(t, err)
	sqlText := string(sqlBytes)
	assert.Contains(t, sqlText, "LIMIT")
	assert.NotContains(t, sqlText, "1000")

	rec := &recordingQuerier{}
	st := &Store{q: rec}
	customerID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	_, err = st.SearchEntities(t.Context(), customerID, "camp", 25, []string{"campaign"})
	require.NoError(t, err)
	assert.LessOrEqual(t, rec.campaignLimit, int32(MaxSearchLimit))
}

type recordingSearcher struct {
	called bool
	items  []ItemDTO
}

func (r *recordingSearcher) SearchEntities(context.Context, uuid.UUID, string, int, []string) ([]ItemDTO, error) {
	r.called = true
	if r.items == nil {
		return []ItemDTO{}, nil
	}
	return r.items, nil
}

func TestStore_SearchEntities_emptyQuerySkipsPG(t *testing.T) {
	t.Parallel()
	rec := &recordingQuerier{}
	st := &Store{q: rec}
	customerID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	items, err := st.SearchEntities(t.Context(), customerID, " ", 25, nil)
	require.NoError(t, err)
	assert.Nil(t, items)
	assert.Equal(t, int32(0), rec.campaignLimit)
}

func TestCampaignRowsToCandidates_statusMapping(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("00000000-0000-4000-8000-000000000099")
	rows := []db.SearchCommandPaletteCampaignsRow{{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		Name:   "Camp Alpha",
		Status: db.CampaignStatusTypeACTIVE,
		UpdatedAt: pgtype.Timestamptz{
			Time:  timeFromPG(pgtype.Timestamptz{}),
			Valid: false,
		},
	}}
	items := campaignRowsToCandidates("camp", rows)
	require.Len(t, items, 1)
	assert.Equal(t, "campaign", items[0].item.Kind)
	assert.True(t, strings.HasPrefix(items[0].item.Href, "/campaigns/"))
}
