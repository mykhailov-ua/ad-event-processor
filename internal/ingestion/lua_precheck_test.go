package ingestion

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnifiedFilter_LuaScriptSlimmed_noDeterministicGates_holdout(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		script string
	}{
		{name: "unified", script: unifiedFilterLua},
		{name: "budget_fast", script: budgetFastLua},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lower := strings.ToLower(tc.script)
			require.NotContains(t, lower, "sismember", "fraud blacklist must run in Go precheck")
			require.NotContains(t, lower, "max_rpd", "ingress RPD must run in Go precheck")
		})
	}
}

func TestUnifiedFilter_LuaScriptSlimmed_ttcGateInGoFlag(t *testing.T) {
	require.Contains(t, unifiedFilterLua, "ttc_in_go")
	require.Contains(t, unifiedFilterLua, "ARGV[34]")
}
