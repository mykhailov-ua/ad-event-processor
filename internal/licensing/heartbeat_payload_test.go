package licensing

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHeartbeatPayload_noForbiddenFields(t *testing.T) {
	forbidden := []string{
		"campaign", "campaign_id", "domain", "domain_id", "ip", "ip_address",
		"hostname", "url", "email", "user_id",
	}
	cases := []any{
		HeartbeatPayload{},
		ActivatePayload{},
	}
	for _, payload := range cases {
		typ := reflect.TypeOf(payload)
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			tag := field.Tag.Get("json")
			name := strings.Split(tag, ",")[0]
			for _, bad := range forbidden {
				require.NotEqual(t, bad, name, "%s must not expose %s", typ.Name(), bad)
			}
		}
	}

	rawHeartbeat, err := json.Marshal(HeartbeatPayload{
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
