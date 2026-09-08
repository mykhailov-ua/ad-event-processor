package track

import (
	"strings"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/require"
)

func TestSafePageStubBody_holdout_noCommercialURL(t *testing.T) {
	body := AppendSafePageStubBody(nil)
	html := string(body)
	require.NotContains(t, html, "<iframe")
	require.NotContains(t, html, "safe.example")
	require.Contains(t, html, `id="aed-mount"`)
	require.Contains(t, html, "/static/track-telemetry.js")
	require.Contains(t, html, "/static/antifraud-telemetry.js")
	require.Contains(t, html, "/track/verify")
}

func TestSafePageStubBody_strictAttestation_usesStealthBundle_holdout(t *testing.T) {
	camp := &domain.Campaign{
		SafePageEnabled:    true,
		AttestationEnabled: true,
		AttestationMode:    domain.AttestationModeStrict,
	}
	body := AppendSafePageStubBodyForCampaign(nil, camp)
	html := string(body)
	require.Contains(t, html, "/static/telemetry-stealth-poc.js")
	require.NotContains(t, html, "/static/antifraud-telemetry.js")
	require.NotContains(t, html, "/track/verify")
	require.Contains(t, html, "aedSensBootstrap")
	require.NotContains(t, html, "WebGLRenderingContext")
}

func TestSafePageHydrator_holdout_serverAuthoritativeUnlock(t *testing.T) {
	src := string(safePageHydratorJS)
	require.NotContains(t, src, "revealFrame")
	require.Contains(t, src, "graftVerifiedHtml")
	require.Contains(t, src, "requestServerUnlock")
	require.True(t, strings.Index(src, "graftVerifiedHtml") < strings.Index(src, "requestServerUnlock"),
		"graft must be defined before server unlock path")
}

func TestSafePageDecoyBody_embedsSafeURL(t *testing.T) {
	body := AppendSafePageDecoyBody(nil, []byte("https://safe.example/decoy"))
	html := string(body)
	require.Contains(t, html, "<iframe")
	require.Contains(t, html, "https://safe.example/decoy")
}
