package management

import (
	"testing"
	"time"

	db "espx/internal/ingestion/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCampaign_validateDaypartHours(t *testing.T) {
	t.Parallel()
	require.NoError(t, validateDaypartHours([]int16{0, 12, 23}))
	require.Error(t, validateDaypartHours([]int16{24}))
}

func TestCampaign_validateSchedule(t *testing.T) {
	t.Parallel()
	start := time.Now()
	end := start.Add(time.Hour)
	require.NoError(t, validateSchedule(&start, &end))
	require.Error(t, validateSchedule(&end, &start))
}

func TestCampaign_resolveScheduleStatus(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)
	assert.Equal(t, db.CampaignStatusTypeACTIVE, resolveScheduleStatus(now, &start, &end))
}

func TestCampaign_countriesOrEmpty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{}, countriesOrEmpty(nil))
	assert.Equal(t, []string{"US"}, countriesOrEmpty([]string{"US"}))
}

func TestCampaign_templateToDTO(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	cust := uuid.New()
	brand := uuid.New()
	now := time.Now()
	row := db.CampaignTemplate{
		ID:          pgtype.UUID{Bytes: id, Valid: true},
		CustomerID:  pgtype.UUID{Bytes: cust, Valid: true},
		Name:        "tpl",
		BudgetLimit: 1_000_000,
		PacingMode:  db.PacingModeTypeEVEN,
		DailyBudget: 100_000,
		Timezone:    "UTC",
		BrandID:     pgtype.UUID{Bytes: brand, Valid: true},
		CreatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
	}
	dto := templateToDTO(row)
	assert.Equal(t, id.String(), dto.ID)
	assert.Equal(t, brand.String(), dto.BrandID)
}

func TestCampaign_creativeToDTO(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	brand := uuid.New()
	now := time.Now()
	row := db.BrandCreative{
		ID:         pgtype.UUID{Bytes: id, Valid: true},
		BrandID:    pgtype.UUID{Bytes: brand, Valid: true},
		Name:       "creative",
		LandingUrl: "https://example.com",
		Weight:     100,
		Status:     "ACTIVE",
		CreatedAt:  pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:  pgtype.Timestamptz{Time: now, Valid: true},
	}
	dto := creativeToDTO(row)
	assert.Equal(t, id.String(), dto.ID)
}

func TestCampaign_DomainMapped(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "campaign", FileDomain("campaign_validate.go"))
}
