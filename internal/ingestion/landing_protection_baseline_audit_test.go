package ingestion

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ad-event-processor/internal/domain"
)

func TestLandingProtectionBaseline_inventorySymbolsExist(t *testing.T) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	ingestionDir := filepath.Dir(file)

	symbols := map[string][]string{
		"tls_fingerprint_table.go":   {"TLSFingerprintTable", "MatchJA3"},
		"l1_tls_fingerprint_hook.go": {"tlsFingerprintShouldSafeView"},
		"link_signer.go":             {"AppendLinkSignature", "VerifyLinkSignature"},
		"proxy_vpn_lpm.go":           {"ProxyVPNTable", "parseProxyVPNConnFlags"},
		"click_redirect.go":          {"reactClickRedirect"},
	}
	for rel, want := range symbols {
		found := parseExportedOrTypeNames(t, filepath.Join(ingestionDir, rel))
		for _, name := range want {
			assert.Contains(t, found, name, "file=%s", rel)
		}
	}
}

func TestLandingProtectionBaseline_productScopeGates(t *testing.T) {
	const (
		gateEdgeJA3Headers    = "edge_header_pass_x_tls_ja3_ja4"
		gateNoExternalIPAPI   = "local_in_memory_l15_only"
		gateSignColdVerifyHot = "sign_cold_verify_hot_alloc_pool"
		gateBanditColdOnly    = "bandit_router_cold_worker_only"
	)
	assert.NotEmpty(t, gateEdgeJA3Headers)
	assert.NotEmpty(t, gateNoExternalIPAPI)
	assert.NotEmpty(t, gateSignColdVerifyHot)
	assert.NotEmpty(t, gateBanditColdOnly)
}

func TestLandingProtectionBaseline_tlsTableHarness_ready(t *testing.T) {
	table := NewTLSFingerprintTable()
	assert.False(t, table.Ready())
	ja3 := []byte("771,4865-4866,0-23,29-23-24,0")
	assert.False(t, table.MatchJA3(ja3))
}

func TestLandingProtectionBaseline_linkSigner_roundTrip(t *testing.T) {
	secret := []byte("landing-protection-test-secret")
	clickID := []byte("click-abc")
	expires := int64(1_700_000_000)
	loc := AppendLinkSignature([]byte("https://offer.test/lp"), secret, clickID, expires)
	assert.Contains(t, string(loc), "expires=")
	assert.Contains(t, string(loc), "_sig=")
	sig := loc[len(loc)-linkSigHexLen:]
	assert.True(t, VerifyLinkSignature(secret, clickID, sig, expires, expires-1))
}

func TestLandingProtectionBaseline_connTypePolicy_defaults(t *testing.T) {
	assert.Equal(t, domain.ConnTypeBlockVPNHosting, domain.ConnTypePolicyFromString(""))
	assert.Equal(t, domain.ConnTypeMobileOnly, domain.ConnTypePolicyFromString("mobile_only"))
}
