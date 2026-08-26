package postback

import (
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
)

func TestClearPayloadBoolFlag_removesPending(t *testing.T) {
	out := clearPayloadBoolFlag([]byte(`{"goal_name":"lead","conversion_validation_pending":true}`), domain.ConversionValidationPendingKey)
	if domain.ConversionValidationPending(out) {
		t.Fatalf("payload %s", out)
	}
	if string(out) != `{"goal_name":"lead"}` {
		t.Fatalf("payload %s", out)
	}
}

func TestNewConversionRejectReprocessor_disabled(t *testing.T) {
	if NewConversionRejectReprocessor(config.ConversionReject{Enabled: false}, nil, nil, nil, nil, nil, nil) != nil {
		t.Fatal("expected nil when disabled")
	}
}

func TestConversionRejectReasonWeight(t *testing.T) {
	if ConversionRejectReasonWeight(ConversionRejectLowTTC) != 45 {
		t.Fatal("weight")
	}
	if ConversionRejectReasonWeight("unknown") != 0 {
		t.Fatal("unknown weight")
	}
}
