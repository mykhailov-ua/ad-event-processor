package fraud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResidentialIntelResult_IsResidentialProxyFarm_holdout(t *testing.T) {
	neg := ResidentialIntelResult{ResidentialProxy: false, VPN: false, Proxy: false}
	assert.False(t, neg.IsResidentialProxyFarm())

	pos := ResidentialIntelResult{ResidentialProxy: true}
	assert.True(t, pos.IsResidentialProxyFarm())

	vpnProxy := ResidentialIntelResult{Proxy: true, VPN: true}
	assert.True(t, vpnProxy.IsResidentialProxyFarm())

	proxyOnly := ResidentialIntelResult{Proxy: true, VPN: false}
	assert.False(t, proxyOnly.IsResidentialProxyFarm())
}
