package management

import (
	"testing"
	"time"

	db "espx/internal/ingestion/sqlc"

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

func TestCampaign_DomainMapped(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "campaign", FileDomain("campaign_validate.go"))
}
