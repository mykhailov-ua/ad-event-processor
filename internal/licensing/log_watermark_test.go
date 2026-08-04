package licensing

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateLogWatermark(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	watermarkApplied = false

	UpdateLogWatermark(&LicenseClaims{
		CustomerName: "Acme Pilot",
		DeploymentID: "dep-123",
		SKU:          "pilot",
	})
	UpdateLogWatermark(&LicenseClaims{
		CustomerName: "Acme Pilot",
		DeploymentID: "dep-123",
	})

	out := buf.String()
	require.Contains(t, out, "license_customer")
	require.Contains(t, out, "Acme Pilot")
	require.Contains(t, out, "dep-123")
}
