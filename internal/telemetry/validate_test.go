package telemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatePayloadJSON_rejectsForbiddenFields(t *testing.T) {
	cases := []string{
		`{"campaign_id":"x"}`,
		`{"domain":"evil.com"}`,
		`{"ip":"1.1.1.1"}`,
		`{"nested":{"user_id":"u1"}}`,
	}
	for _, raw := range cases {
		require.Error(t, ValidatePayloadJSON([]byte(raw)), raw)
	}
}

func TestValidatePayloadJSON_allowsPulseShape(t *testing.T) {
	raw := []byte(`{
		"schema_version":1,
		"deployment_id":"dep",
		"binary_version":"1.0.0",
		"window_sec":3600,
		"accepted_events":10,
		"rejected_events":2,
		"peak_rps":100
	}`)
	require.NoError(t, ValidatePayloadJSON(raw))
}
