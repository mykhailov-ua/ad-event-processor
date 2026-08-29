package ingest

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRPBaseline_inventorySymbolsExist(t *testing.T) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	ingestionDir := filepath.Dir(file)

	symbols := map[string][]string{
		"click_redirect.go":               {"reactClickRedirect"},
		"click_proxy.go":                  {"clickProxyDeliver"},
		"safe_page.go":                    {"resolveSafePageLanding", "writeSafePageStubResponse"},
		"safe_page_verify.go":             {"reactTrackVerify"},
		"attestation_token.go":            {"MintAttestationToken", "ConfigureAttestation"},
		"cidr_lpm.go":                     {"CIDRTable"},
		"landing_tls_fingerprint_hook.go": {"tlsFingerprintShouldSafeView"},
	}
	for rel, want := range symbols {
		found := parseExportedOrTypeNames(t, filepath.Join(ingestionDir, rel))
		for _, name := range want {
			assert.Contains(t, found, name, "file=%s", rel)
		}
	}
}
