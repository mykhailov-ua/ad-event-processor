package reports

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFraudEvidencePack_layerDesyncConsistentWithDrilldown_holdout(t *testing.T) {
	t.Parallel()
	pack := FraudEvidencePackDTO{
		FraudEvents: []FraudEvidenceFraudRowDTO{{
			FraudReason:      "tcp_syn_os_mismatch,tls_ja4_mismatch",
			LayerDesyncCount: 2,
			FraudScore:       70,
		}},
	}
	sig := aggregateFraudEvidenceSignals(pack.FraudEvents)
	require.Equal(t, uint8(2), sig.MaxLayerDesyncCount)
	assert.GreaterOrEqual(t, sig.MaxLayerDesyncCount, uint8(2))
}
