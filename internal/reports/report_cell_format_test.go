package reports

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatReportCellValue_scalar(t *testing.T) {
	t.Parallel()
	require.Equal(t, "42", FormatReportCellValue(int64(42)))
	require.Equal(t, "ok", FormatReportCellValue("ok"))
	require.Equal(t, "true", FormatReportCellValue(true))
}

func TestFormatReportCellValue_object_holdout(t *testing.T) {
	t.Parallel()
	got := FormatReportCellValue(map[string]any{"a": 1})
	require.Equal(t, `{"a":1}`, got)
}

func TestNormalizeReportMapRows_stringifiesCompareCell(t *testing.T) {
	t.Parallel()
	rows := []map[string]any{{
		"campaign_id": "c1",
		"compare": map[string]any{
			"clicks_delta": int64(3),
		},
	}}
	out := NormalizeReportMapRows(rows)
	require.Equal(t, "c1", out[0]["campaign_id"])
	compare, ok := out[0]["compare"].(string)
	require.True(t, ok)
	require.Contains(t, compare, "clicks_delta")
}
