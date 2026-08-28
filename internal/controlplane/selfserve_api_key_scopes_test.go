package controlplane

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSelfServeAPIKeyScopes_rejectsOpsScope_holdout(t *testing.T) {
	t.Parallel()
	_, err := validateSelfServeAPIKeyScopes([]string{"audit:read"})
	require.Error(t, err)
}

func TestValidateSelfServeAPIKeyScopes_defaultsRead(t *testing.T) {
	t.Parallel()
	got, err := validateSelfServeAPIKeyScopes(nil)
	require.NoError(t, err)
	require.Equal(t, []string{"campaigns:read"}, got)
}
