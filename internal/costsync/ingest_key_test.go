package costsync

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIngestKey_pipeDelimitedFormat(t *testing.T) {
	customerID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	campaignID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	date := time.Date(2026, 7, 1, 15, 30, 0, 0, time.FixedZone("MSK", 3*3600))

	got := IngestKey(customerID, campaignID, date, "facebook", "ad-1", LineTypeSpend)
	want := customerID.String() + "|" + campaignID.String() + "|" +
		date.Format("2006-01-02") + "|" + "facebook" + "|" + "ad-1" + "|" + string(LineTypeSpend)
	require.Equal(t, want, got)
	require.Contains(t, got, "2026-07-01")
}
