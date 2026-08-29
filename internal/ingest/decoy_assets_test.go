package ingest

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveUnifiedFilterLua_ignoresDecoyEmbed(t *testing.T) {
	decoy := decoyUnifiedFilterEmbed()
	require.NotEmpty(t, decoy)
	require.Contains(t, decoy, "unified_filter_check")
	require.NotEqual(t, unifiedFilterLua, decoy)

	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_MODE", "dev")
	src, err := resolveUnifiedFilterLuaSource()
	require.NoError(t, err)
	require.Equal(t, unifiedFilterLua, src)
	require.NotEqual(t, decoy, src)
}

func TestDecoyUnifiedFilter_notReferencedBySealedResolver(t *testing.T) {
	raw, err := os.ReadFile("filter_engine_bundle.go")
	require.NoError(t, err)
	body := string(raw)
	require.NotContains(t, body, "decoyUnifiedFilterEmbed")
	require.NotContains(t, body, "decoy_unified_filter.lua")
}

func TestDecoyUnifiedFilter_distinctFromProductionEmbed(t *testing.T) {
	require.False(t, strings.HasPrefix(decoyUnifiedFilterEmbed(), unifiedFilterLua[:min(32, len(unifiedFilterLua))]))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
