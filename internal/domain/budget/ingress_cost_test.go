package budget

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseIngressCostConfigJSON(t *testing.T) {
	cfg := ParseIngressCostConfigJSON([]byte(`{"param":"cost","scale":"decimal","max_micro":5000000,"policy":"ignore"}`))
	require.Equal(t, IngressCostParamCost, cfg.Param)
	require.False(t, cfg.ScaleMicro)
	require.Equal(t, int64(5_000_000), cfg.MaxMicro)
	require.Equal(t, IngressCostPolicyIgnore, cfg.Policy)
	require.True(t, cfg.Enabled())
}

func TestIngressCostConfigMarshalJSON_disabled(t *testing.T) {
	raw, err := IngressCostConfig{}.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, "{}", string(raw))
}
