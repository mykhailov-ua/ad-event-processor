package platformadmin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCloudflareRecordTypeForTarget(t *testing.T) {
	t.Parallel()
	require.Equal(t, "A", cloudflareRecordTypeForTarget("203.0.113.1"))
	require.Equal(t, "CNAME", cloudflareRecordTypeForTarget("ingress.example.com"))
	require.Equal(t, "AAAA", cloudflareRecordTypeForTarget("2001:db8::1"))
}
