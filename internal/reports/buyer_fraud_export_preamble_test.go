package reports

import (
	"bytes"
	"context"
	"encoding/csv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataFreshnessFromClickHouse_nilQueryStale_holdout(t *testing.T) {
	t.Parallel()
	dto := DataFreshnessFromClickHouse(context.Background(), nil)
	assert.True(t, dto.Stale)
}

func TestWriteBuyerFraudExportPreamble_signalsDegradedWhenStale_holdout(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	require.NoError(t, WriteBuyerFraudExportPreamble(w, DataFreshnessDTO{Stale: true}))
	w.Flush()
	content := buf.String()
	assert.Contains(t, content, "signals_degraded")
	assert.Contains(t, content, "true")
}

func TestWriteBuyerFraudExportPreamble_freshOmitsDegradedRow(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	require.NoError(t, WriteBuyerFraudExportPreamble(w, DataFreshnessDTO{Stale: false}))
	w.Flush()
	assert.NotContains(t, buf.String(), "# signals_degraded")
}
