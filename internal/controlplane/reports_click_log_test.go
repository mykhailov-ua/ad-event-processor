package controlplane

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNarrowCampaignIDs_filter(t *testing.T) {
	campA := uuid.New()
	campB := uuid.New()
	got := narrowCampaignIDs([]uuid.UUID{campA, campB}, campB, true)
	require.Equal(t, []uuid.UUID{campB}, got)
	require.Nil(t, narrowCampaignIDs([]uuid.UUID{campA}, campB, true))
}

func TestClickLogPayloadFields_extractsRevenue(t *testing.T) {
	sub1, country, status, goal, revenue := clickLogPayloadFields(`{"sub1":"aff","country":"US","status":"lead","goal_name":"lead","revenue_micro":1500000}`)
	require.Equal(t, "aff", sub1)
	require.Equal(t, "US", country)
	require.Equal(t, "lead", status)
	require.Equal(t, "lead", goal)
	require.Equal(t, int64(1_500_000), revenue)
}

func TestClickLogPayloadFields_empty(t *testing.T) {
	sub1, country, status, goal, revenue := clickLogPayloadFields("")
	require.Equal(t, "", sub1)
	require.Equal(t, "", country)
	require.Equal(t, "", status)
	require.Equal(t, "", goal)
	require.Equal(t, int64(0), revenue)
}
