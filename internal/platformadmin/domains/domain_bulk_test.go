package domains

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeBulkHostnames_csvAndDedup(t *testing.T) {
	t.Parallel()
	hosts, err := normalizeBulkHostnames(
		[]string{"a.example", "A.example"},
		"hostname\nb.example\nc.example,b.example",
	)
	require.NoError(t, err)
	require.Equal(t, []string{"a.example", "b.example", "c.example"}, hosts)
}

func TestNormalizeBulkHostnames_maxExceeded(t *testing.T) {
	t.Parallel()
	raw := make([]string, domainBulkMaxHostnames+1)
	for i := range raw {
		raw[i] = fmt.Sprintf("host-%d.example", i)
	}
	_, err := normalizeBulkHostnames(raw, "")
	require.Error(t, err)
}

func TestNormalizeBulkHostnames_empty(t *testing.T) {
	t.Parallel()
	_, err := normalizeBulkHostnames(nil, "  \n")
	require.Error(t, err)
}
