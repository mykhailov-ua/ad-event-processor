package postback

const (
	ConversionRejectNoClick      = "conversion_no_click"
	ConversionRejectLowTTC       = "conversion_low_ttc"
	ConversionRejectDuplicate    = "conversion_duplicate"
	ConversionRejectIPDrift      = "conversion_ip_drift"
	ConversionRejectDatacenterIP = "conversion_datacenter_ip"
)

var ConversionRejectReasonWeights = map[string]uint8{
	ConversionRejectNoClick:      50,
	ConversionRejectLowTTC:       45,
	ConversionRejectDuplicate:    55,
	ConversionRejectIPDrift:      40,
	ConversionRejectDatacenterIP: 45,
}

func ConversionRejectReasonWeight(code string) uint8 {
	if w, ok := ConversionRejectReasonWeights[code]; ok {
		return w
	}
	return 0
}
