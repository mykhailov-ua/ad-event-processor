package platformconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultValidate(t *testing.T) {
	cfg := Default()
	require.NoError(t, cfg.Validate())
}

func TestParseAndMarshalRoundTrip(t *testing.T) {
	cfg := Default()
	cfg.TrackingDomain = "trk.example.com"
	cfg.Stripe.Enabled = true
	cfg.Stripe.SecretKey = "sk_test_abc"

	raw, err := Marshal(cfg)
	require.NoError(t, err)

	got, err := Parse(raw)
	require.NoError(t, err)
	assert.Equal(t, cfg.TrackingDomain, got.TrackingDomain)
	assert.True(t, got.Stripe.Enabled)
	assert.Equal(t, cfg.Stripe.SecretKey, got.Stripe.SecretKey)
}

func TestPatchPreservesSecrets(t *testing.T) {
	base := Default()
	base.Stripe.Enabled = true
	base.Stripe.SecretKey = "sk_live_secret"

	patch := Patch{
		TrackingDomain: strPtr("trk.new.com"),
		Stripe: &StripePatch{
			Enabled: boolPtr(true),
		},
	}
	got, err := patch.Apply(base)
	require.NoError(t, err)
	assert.Equal(t, "trk.new.com", got.TrackingDomain)
	assert.Equal(t, "sk_live_secret", got.Stripe.SecretKey)
}

func TestRestartRequiredFields(t *testing.T) {
	before := Default()
	after := before
	after.IngressSchema = IngressOpenRTB3
	fields := RestartRequiredFields(before, after)
	assert.Contains(t, fields, "ingress_schema")
}

func TestPublicRedactsSecrets(t *testing.T) {
	cfg := Default()
	cfg.Stripe.SecretKey = "sk_test_12345678"
	view := Public(cfg, true, nil)
	assert.Empty(t, view.Config.Stripe.SecretKey)
	assert.Equal(t, "****5678", view.Secrets.StripeSecretKey)
}

func TestClickURLTemplate(t *testing.T) {
	assert.Equal(t, "https://trk.example.com/click?campaign_id={campaign_id}&sub1={sub1}",
		ClickURLTemplate("trk.example.com"))
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
