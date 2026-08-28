package controlplane

import (
	"ad-event-processor/internal/campaign"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSelfServeAPIKeyScopes_rejectsOpsScope_holdout(t *testing.T) {
	t.Parallel()
	_, err := campaign.ValidateSelfServeAPIKeyScopes([]string{"audit:read"})
	require.Error(t, err)
}

func TestValidateSelfServeAPIKeyScopes_defaultsRead(t *testing.T) {
	t.Parallel()
	got, err := campaign.ValidateSelfServeAPIKeyScopes(nil)
	require.NoError(t, err)
	require.Equal(t, []string{"campaigns:read"}, got)
}
