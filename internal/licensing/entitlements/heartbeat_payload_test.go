package entitlements_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	verify "ad-event-processor/internal/licensing/verify"

	"github.com/stretchr/testify/require"
)

func TestHeartbeatPayload_noForbiddenFields(t *testing.T) {
	forbidden := []string{
		"campaign", "campaign_id", "domain", "domain_id", "ip", "ip_address",
		"hostname", "url", "email", "user_id",
	}
	cases := []any{
		verify.HeartbeatPayload{},
		verify.ActivatePayload{},
	}
	for _, payload := range cases {
		typ := reflect.TypeOf(payload)
		for i := range typ.NumField() {
			field := typ.Field(i)
			tag := field.Tag.Get("json")
			name := strings.Split(tag, ",")[0]
			for _, bad := range forbidden {
				require.NotEqual(t, bad, name, "%s must not expose %s", typ.Name(), bad)
			}
		}
	}

	rawHeartbeat, err := json.Marshal(verify.HeartbeatPayload{
		LicenseKey:    "key",
		DeploymentID:  "dep",
		Fingerprint:   "fp",
		Version:       "1.0.0",
		UptimeSeconds: 42,
	})
	require.NoError(t, err)
	var heartbeatMap map[string]any
	require.NoError(t, json.Unmarshal(rawHeartbeat, &heartbeatMap))
	for key := range heartbeatMap {
		for _, bad := range forbidden {
			require.NotContains(t, strings.ToLower(key), bad)
		}
	}
}
