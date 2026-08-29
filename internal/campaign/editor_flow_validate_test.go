package campaign

import (
	"testing"

	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCampaignFlowValidateResponse_weightSumUnder100Fails_holdout(t *testing.T) {
	t.Parallel()
	resp := BuildCampaignFlowValidateResponse([]FlowPathDTO{
		{Weight: 50, Landers: []FlowPathLanderRef{{LanderID: uuid.New(), Weight: 1}}, Offers: []FlowPathOfferRef{{OfferID: uuid.New(), Weight: 1}}},
		{Weight: 49, Landers: []FlowPathLanderRef{{LanderID: uuid.New(), Weight: 1}}, Offers: []FlowPathOfferRef{{OfferID: uuid.New(), Weight: 1}}},
	})
	assert.False(t, resp.Valid)
	require.NotEmpty(t, resp.PathErrors)
	assert.Equal(t, "weight_sum", resp.PathErrors[len(resp.PathErrors)-1].Code)
}

func TestBuildCampaignFlowValidateResponse_valid100(t *testing.T) {
	t.Parallel()
	resp := BuildCampaignFlowValidateResponse([]FlowPathDTO{
		{Weight: 60, Landers: []FlowPathLanderRef{{LanderID: uuid.New(), Weight: 1}}, Offers: []FlowPathOfferRef{{OfferID: uuid.New(), Weight: 1}}},
		{Weight: 40, Landers: []FlowPathLanderRef{{LanderID: uuid.New(), Weight: 1}}, Offers: []FlowPathOfferRef{{OfferID: uuid.New(), Weight: 1}}},
	})
	assert.True(t, resp.Valid)
}

func TestBuildCampaignFlowValidateResponse_missingLandersFails_holdout(t *testing.T) {
	t.Parallel()
	resp := BuildCampaignFlowValidateResponse([]FlowPathDTO{
		{Weight: 100, Offers: []FlowPathOfferRef{{OfferID: uuid.New(), Weight: 1}}},
	})
	assert.False(t, resp.Valid)
	require.NotEmpty(t, resp.PathErrors)
	assert.Equal(t, "invalid_shape", resp.PathErrors[0].Code)
}

func TestPreviewCampaignMacros_resolvesRedirectMacros(t *testing.T) {
	t.Parallel()
	campaign := CampaignDTO{
		ID:        "camp-1",
		TargetURL: "https://offer.example/lp?cid={click_id}&sub1={sub1}",
	}
	preview, err := previewCampaignMacros(campaign, MacroPreviewRequestDTO{Sub1: "alpha"}, false)
	require.NoError(t, err)
	assert.Contains(t, preview.ResolvedClickURL, "preview-click-id")
	assert.Contains(t, preview.ResolvedClickURL, "alpha")
}

func TestPreviewCampaignMacros_unresolvedPresetMacro(t *testing.T) {
	t.Parallel()
	campaign := CampaignDTO{
		ID:        "camp-1",
		TargetURL: "https://offer.example/?x={{adset.id}}",
	}
	preview, err := previewCampaignMacros(campaign, MacroPreviewRequestDTO{}, false)
	require.NoError(t, err)
	require.NotEmpty(t, preview.UnresolvedMacros)
}

func TestPreviewCampaignMacros_resolvesCampaignId(t *testing.T) {
	t.Parallel()
	campaign := CampaignDTO{
		ID:        "camp-1",
		TargetURL: "https://offer.example/click?cid={{campaign.id}}&sub1={{sub1}}",
	}
	preview, err := previewCampaignMacros(campaign, MacroPreviewRequestDTO{Sub1: "alpha"}, false)
	require.NoError(t, err)
	assert.Contains(t, preview.ResolvedClickURL, "camp-1")
	assert.Contains(t, preview.ResolvedClickURL, "alpha")
}

func TestPreviewCampaignMacros_maskedRedactsOfferURL(t *testing.T) {
	t.Parallel()
	campaign := CampaignDTO{ID: "camp-1", TargetURL: "https://secret.offer/track"}
	preview, err := previewCampaignMacros(campaign, MacroPreviewRequestDTO{}, true)
	require.NoError(t, err)
	assert.Equal(t, "[redacted-offer-url]", preview.ResolvedClickURL)
}

func TestBuildCampaignConflictResponse_includesServerRevision(t *testing.T) {
	t.Parallel()
	current := CampaignDTO{ID: "c1", UpdatedAt: "2026-08-27T10:00:00Z", Revision: "2026-08-27T10:00:00Z", Name: "Live"}
	name := "Draft"
	conflict := buildCampaignConflictResponse(current, PatchCampaignRequest{Name: &name})
	assert.Equal(t, "2026-08-27T10:00:00Z", conflict.ServerRevision)
	assert.Contains(t, conflict.ConflictFields, "name")
}

func TestScrubCustomerFraudEvidencePack_redactsPII_holdout(t *testing.T) {
	t.Parallel()
	pack := reports.FraudEvidencePackDTO{
		Timeline: []reports.FraudEvidenceTimelineRowDTO{{Country: "US", Sub1: "secret", PlacementID: "pl-1"}},
		FraudEvents: []reports.FraudEvidenceFraudRowDTO{{
			FraudReason: "tcp_syn_os_mismatch,tls_ja4_mismatch",
			PlacementID: "pl-1",
		}},
	}
	scrubbed := reports.ScrubCustomerFraudEvidencePack(pack)
	assert.Empty(t, scrubbed.Timeline[0].Country)
	assert.Empty(t, scrubbed.Timeline[0].Sub1)
	assert.Empty(t, scrubbed.FraudEvents[0].PlacementID)
	assert.NotEqual(t, "tcp_syn_os_mismatch,tls_ja4_mismatch", scrubbed.FraudEvents[0].FraudReason)
	secret := []byte("customer-evidence-secret")
	signed, err := reports.BuildSignedFraudEvidencePack(secret, scrubbed)
	require.NoError(t, err)
	require.NoError(t, reports.VerifyFraudEvidencePackSignature(secret, signed))
}
