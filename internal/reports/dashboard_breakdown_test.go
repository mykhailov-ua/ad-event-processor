package reports

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFlowEntityBreakdownQuery_whitelist_holdout(t *testing.T) {
	t.Parallel()
	require.Panics(t, func() {
		_ = flowEntityBreakdownQuery("sub1")
	})
	require.Contains(t, flowEntityBreakdownQuery("lander_id"), "lander_id")
	require.Contains(t, flowEntityBreakdownQuery("offer_id"), "offer_id")
}
