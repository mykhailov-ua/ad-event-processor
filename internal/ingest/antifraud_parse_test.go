package ingest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAntifraud_holdout(t *testing.T) {
	body := []byte(`{"campaign_id":"00000000-0000-4000-8000-000000000001","type":"conversion","antifraud":{"dwell_ms":3200,"pointer_cv_milli":22,"scroll_cv_milli":18,"scroll_jerk_milli":11,"trusted_ratio_milli":950,"webdriver":1,"automation_leak":16,"canvas_hash":"deadbeef","rtt_samples":[12,14,58,13]}}`)
	req := &TrackRequest{}
	require.NoError(t, ParseTrackRequestJSON(req, body))
	require.Equal(t, uint8(1), req.AntifraudSet)
	require.Equal(t, uint32(3200), req.AntifraudSnapshot.DwellMs)
	require.Equal(t, uint16(22), req.AntifraudSnapshot.PointerCVMilli)
	require.Equal(t, uint8(1), req.AntifraudSnapshot.Webdriver)
	require.Equal(t, uint8(4), req.AntifraudSnapshot.RTTSampleCount)
	require.Equal(t, uint16(58), req.AntifraudSnapshot.RTTSamples[2])
}

func TestParseAntifraud_missing_ok(t *testing.T) {
	body := []byte(`{"campaign_id":"00000000-0000-4000-8000-000000000001","type":"click"}`)
	req := &TrackRequest{}
	require.NoError(t, ParseTrackRequestJSON(req, body))
	require.Equal(t, uint8(0), req.AntifraudSet)
}
