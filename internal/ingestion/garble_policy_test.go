package ingestion

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGarblePolicy_ingestionPackageIgnored(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("garble_policy.go")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(b), "//garble:ignore"))
}
