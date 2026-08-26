package costsync

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClickMatchColumn(t *testing.T) {
	t.Parallel()
	col, err := clickMatchColumn("placement_id")
	require.NoError(t, err)
	require.Equal(t, "placement_id", col)

	col, err = clickMatchColumn("sub1")
	require.NoError(t, err)
	require.Equal(t, "sub1", col)

	_, err = clickMatchColumn("sub3")
	require.Error(t, err)
}

func TestValidSyncIntervalMinutes(t *testing.T) {
	t.Parallel()
	require.True(t, ValidSyncIntervalMinutes(15))
	require.True(t, ValidSyncIntervalMinutes(1440))
	require.False(t, ValidSyncIntervalMinutes(5))
	require.False(t, ValidSyncIntervalMinutes(0))
}

func TestParseTokenMapping_defaults(t *testing.T) {
	t.Parallel()
	m := ParseTokenMapping(nil)
	require.Equal(t, "placement_id", m.PlacementField)
	require.Equal(t, "ad_id", m.NetworkObject)
	require.Equal(t, AttributionModeToken, m.AttributionMode)
}

func TestParseTokenMapping_spread(t *testing.T) {
	t.Parallel()
	m := ParseTokenMapping([]byte(`{"placement_field":"sub1","attribution_mode":"spread"}`))
	require.Equal(t, "sub1", m.PlacementField)
	require.Equal(t, AttributionModeSpread, m.AttributionMode)
}
