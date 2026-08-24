package integrationschema_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ad-event-processor/internal/integrationschema"
	"ad-event-processor/internal/postback"
	"github.com/stretchr/testify/require"
)

func TestIntegrationSchema_ParseBundledYAML(t *testing.T) {
	root := filepath.Join("..", "..", "deploy", "schemas")
	for _, file := range []string{
		"inbound_tokens.v1.yaml",
		"outbound_postback.v1.yaml",
		"status_mapping.v1.yaml",
	} {
		raw, err := os.ReadFile(filepath.Join(root, file))
		require.NoError(t, err, file)
		kind, parsed, err := integrationschema.ParseDocument(raw)
		require.NoError(t, err, file)
		require.NotNil(t, parsed, file)
		require.NotEmpty(t, kind, file)
	}
}

func TestIntegrationSchema_InvalidUnknownField(t *testing.T) {
	_, _, err := integrationschema.ParseDocument([]byte(`{
		"version": 1,
		"tokens": [{"name":"gclid","query_key":"gclid","max_len":256}],
		"unexpected": true
	}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown")
}

func TestIntegrationSchema_OutboundUndeclaredPlaceholder(t *testing.T) {
	_, _, err := integrationschema.ParseDocument([]byte(`{
		"version": 1,
		"url_template": "https://x.test/pb?c={click_id}&z={unknown_macro}",
		"placeholders": ["click_id"]
	}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "undeclared")
}

func TestIntegrationSchema_StatusMappingRoundTrip(t *testing.T) {
	_, parsed, err := integrationschema.ParseDocument([]byte(`{
		"version": 1,
		"status_map": {"lead": "pending", "sale": "approved"}
	}`))
	require.NoError(t, err)
	s := parsed.(*integrationschema.StatusMappingSchema)
	mapped, ok := integrationschema.MapAffiliateStatus(s, "sale")
	require.True(t, ok)
	require.Equal(t, "approved", mapped)
}

func TestIntegrationSchema_InboundClickQueryRoundTrip(t *testing.T) {
	_, parsed, err := integrationschema.ParseDocument([]byte(`{
		"version": 1,
		"tokens": [
			{"name":"gclid","query_key":"gclid","max_len":256},
			{"name":"sub1","query_key":"sub1","max_len":64}
		],
		"macros": [
			{"name":"campaign_id","source":"query","key":"campaign_id","required":true},
			{"name":"click_id","source":"query","key":"click_id","required":true}
		]
	}`))
	require.NoError(t, err)
	inbound := parsed.(*integrationschema.InboundTokensSchema)
	q, err := integrationschema.BuildInboundClickQuery(inbound, map[string]string{
		"campaign_id": "550e8400-e29b-41d4-a716-446655440000",
		"click_id":    "clk-m6-rt",
		"gclid":       "gclid-abc",
		"sub1":        "facebook",
	})
	require.NoError(t, err)
	require.Contains(t, q, "campaign_id=550e8400")
	require.Contains(t, q, "click_id=clk-m6-rt")
	require.Contains(t, q, "gclid=gclid-abc")
	require.Contains(t, q, "sub1=facebook")

	tpl := postback.ParseTemplate("https://aff.example.com/pb?click_id={click_id}&sub1={sub1}")
	var scratch [postback.MaxRenderedURLLen]byte
	ctx := postback.EventContext{ClickID: "clk-m6-rt"}
	ctx.SubIDs[0] = "facebook"
	rendered := string(tpl.RenderStack(&ctx, &scratch))
	require.True(t, strings.Contains(rendered, "clk-m6-rt"))
	require.True(t, strings.Contains(rendered, "facebook"))
	t.Logf("fault_proof fault=integration_schema_roundtrip harness=unit click_id=clk-m6-rt sub1=facebook")
}

func TestIntegrationSchema_TokenMaxLenRejected(t *testing.T) {
	_, parsed, err := integrationschema.ParseDocument([]byte(`{
		"version": 1,
		"tokens": [{"name":"gclid","query_key":"gclid","max_len":8}],
		"macros": []
	}`))
	require.NoError(t, err)
	inbound := parsed.(*integrationschema.InboundTokensSchema)
	_, err = integrationschema.BuildInboundClickQuery(inbound, map[string]string{
		"gclid": "123456789",
	})
	require.Error(t, err)
}
