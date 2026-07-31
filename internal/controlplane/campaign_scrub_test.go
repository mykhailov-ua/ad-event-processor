package controlplane

import (
	"encoding/json"
	"testing"

	"espx/internal/controlplane/authz"

	"github.com/stretchr/testify/assert"
)

func TestCampaignResponse_Scrub(t *testing.T) {
	dto := CampaignDTO{
		ID:              "c1",
		Name:            "test",
		TargetURL:       "https://secret.example/landing",
		CreativePayload: json.RawMessage(`{"img":"x"}`),
		ReferrerFilter:  "ref.example",
	}
	full := dto.Scrub(authz.MaskFull)
	assert.Equal(t, "https://secret.example/landing", full.TargetURL)

	masked := dto.Scrub(authz.MaskMasked)
	assert.Empty(t, masked.TargetURL)
	assert.Nil(t, masked.CreativePayload)
	assert.Empty(t, masked.ReferrerFilter)
	assert.Equal(t, "test", masked.Name)
}
