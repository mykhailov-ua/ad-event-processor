package controlplane

import (
	"testing"

	"ad-event-processor/internal/commandpalette"
	"ad-event-processor/internal/reports"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandPalette_reportCatalogParity_holdout(t *testing.T) {
	t.Parallel()
	paletteKeys := commandpalette.ReportNavKeys()
	require.Len(t, paletteKeys, len(reports.ReportCatalogEntries))
	catalogKeys := make(map[string]struct{}, len(reports.ReportCatalogEntries))
	for _, row := range reports.ReportCatalogEntries {
		catalogKeys[row.Key] = struct{}{}
	}
	for _, key := range paletteKeys {
		_, ok := catalogKeys[key]
		assert.True(t, ok, "palette report key %q missing from reports.ReportCatalogEntries", key)
	}
}
