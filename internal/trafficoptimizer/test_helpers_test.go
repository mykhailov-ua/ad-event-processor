package trafficoptimizer

import (
	"bytes"
	"encoding/json"
	"testing"
)

func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(b)
}

func decodeJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
