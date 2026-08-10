package ingestion

import (
	"testing"
)

func FuzzParseTrackJSON(f *testing.F) {
	f.Add([]byte(`{"type":"click","campaign_id":"550e8400-e29b-41d4-a716-446655440000"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"a":1,"b":2}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var req TrackRequest
		fuzzNoPanic(t, "ParseTrackRequestJSON", func() {
			_ = ParseTrackRequestJSON(&req, data)
		})
		fuzzNoPanic(t, "ParseTrackRequestJSONOpt", func() {
			_ = ParseTrackRequestJSONOpt(&req, data)
		})
	})
}
