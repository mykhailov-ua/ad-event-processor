package billingadmin

import (
	"testing"

	"ad-event-processor/internal/licensing"

	"github.com/stretchr/testify/require"
)

func TestExportChunkMaxBytes_pilotDisabled(t *testing.T) {
	got := ExportChunkMaxBytes(licensing.Limits{MaxExportChunkBytes: 0}, licensing.StateActive, true)
	require.True(t, ExportDisabled(got))
}

func TestExportChunkMaxBytes_starterCap(t *testing.T) {
	got := ExportChunkMaxBytes(licensing.Limits{MaxExportChunkBytes: 5 << 20}, licensing.StateActive, true)
	require.Equal(t, 5<<20, got)
	require.False(t, ExportDisabled(got))
}

func TestExportChunkMaxBytes_unlicensedDefault(t *testing.T) {
	got := ExportChunkMaxBytes(licensing.Limits{}, licensing.StateExpired, false)
	require.Equal(t, DefaultExportChunkMaxBytes, got)
	require.False(t, ExportDisabled(got))
}
