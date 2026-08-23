package edge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalFraudQuarantinePayload_dedupes(t *testing.T) {
	raw, err := MarshalFraudQuarantinePayload([]string{"203.0.113.1", "203.0.113.1", "203.0.113.2"})
	require.NoError(t, err)
	ips, err := ParseFraudQuarantinePayload(raw)
	require.NoError(t, err)
	assert.Equal(t, []string{"203.0.113.1", "203.0.113.2"}, ips)
}

func TestParseFraudQuarantinePayload_legacySingleIP(t *testing.T) {
	ips, err := ParseFraudQuarantinePayload("203.0.113.10")
	require.NoError(t, err)
	assert.Equal(t, []string{"203.0.113.10"}, ips)
}
