package config

type ConversionReject struct {
	Enabled                bool
	MinTTCDurationMs       int
	RejectNoClick          bool
	RejectLowTTC           bool
	RejectDuplicate        bool
	RejectIPDrift          bool
	RejectDatacenterIP     bool
	ReprocessEnabled       bool
	ReprocessIntervalMin   int
	ReprocessLookbackHours int
}

func (c *Config) ConversionSmartRejectEnabled() bool {
	return c != nil && c.ConversionReject.Enabled
}
