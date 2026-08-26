package postback

import (
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
)

func mergeConversionRejectConfig(global config.ConversionReject, rules domain.ConversionRejectRules) config.ConversionReject {
	out := global
	if rules.Enabled != nil {
		out.Enabled = *rules.Enabled
	}
	if rules.MinTTCDurationMs != nil {
		out.MinTTCDurationMs = *rules.MinTTCDurationMs
	}
	if rules.RejectNoClick != nil {
		out.RejectNoClick = *rules.RejectNoClick
	}
	if rules.RejectLowTTC != nil {
		out.RejectLowTTC = *rules.RejectLowTTC
	}
	if rules.RejectDuplicate != nil {
		out.RejectDuplicate = *rules.RejectDuplicate
	}
	if rules.RejectIPDrift != nil {
		out.RejectIPDrift = *rules.RejectIPDrift
	}
	if rules.RejectDatacenterIP != nil {
		out.RejectDatacenterIP = *rules.RejectDatacenterIP
	}
	return out
}
