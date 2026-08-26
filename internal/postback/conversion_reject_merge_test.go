package postback

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

func TestMergeConversionRejectConfig(t *testing.T) {
	global := config.ConversionReject{
		Enabled:            true,
		MinTTCDurationMs:   3000,
		RejectNoClick:      true,
		RejectLowTTC:       true,
		RejectDuplicate:    true,
		RejectIPDrift:      true,
		RejectDatacenterIP: false,
	}
	disabled := false
	ttc := 5000
	dc := true
	rules := domain.ConversionRejectRules{
		Enabled:            &disabled,
		MinTTCDurationMs:   &ttc,
		RejectDatacenterIP: &dc,
	}
	out := mergeConversionRejectConfig(global, rules)
	if out.Enabled {
		t.Fatal("enabled")
	}
	if out.MinTTCDurationMs != 5000 {
		t.Fatal("ttc")
	}
	if !out.RejectDatacenterIP {
		t.Fatal("dc")
	}
	if !out.RejectNoClick {
		t.Fatal("no_click inherit")
	}
}

func TestConversionReject_campaignRulesDisabled(t *testing.T) {
	campID := uuid.New()
	disabled := false
	reader := &stubConversionCampaignReader{
		rejectRules: map[uuid.UUID]domain.ConversionRejectRules{
			campID: {Enabled: &disabled},
		},
	}
	applier := NewConversionRejectApplier(conversionRejectCfg(), nil, reader, nil)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "",
		Payload:    []byte(`{"goal_name":"lead"}`),
	}
	applier.ApplyBatch(context.Background(), []*domain.Event{evt})
	if evt.FraudReason != "" {
		t.Fatalf("reason %q", evt.FraudReason)
	}
}
