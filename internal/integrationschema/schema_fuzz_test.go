package integrationschema

import "testing"

func FuzzIntegrationSchemaParse(f *testing.F) {
	f.Add([]byte(`{"version":1,"tokens":[{"name":"gclid","query_key":"gclid","max_len":32}]}`))
	f.Add([]byte("version: 1\nstatus_map:\n lead: pending\n"))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > MaxBodyBytes {
			raw = raw[:MaxBodyBytes]
		}
		_, _, _ = ParseDocument(raw)
	})
}
