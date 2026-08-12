package controlplane

import (
	"testing"
	"time"

	db "github.com/bidshard/ad-event-processor/internal/domain/db"

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

	futureStart := now.Add(time.Hour)
	assert.Equal(t, db.CampaignStatusTypePAUSED, resolveScheduleStatus(now, &futureStart, nil))

	pastEnd := now.Add(-time.Hour)
	pastStart := now.Add(-2 * time.Hour)
	assert.Equal(t, db.CampaignStatusTypePAUSED, resolveScheduleStatus(now, &pastStart, &pastEnd))
}

func TestCampaign_countriesOrEmpty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{}, countriesOrEmpty(nil))
	assert.Equal(t, []string{"US"}, countriesOrEmpty([]string{"US"}))
}
