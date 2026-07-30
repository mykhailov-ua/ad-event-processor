package config

const (
	IngressSchemaOpenRTB3   = "openrtb_3"
	IngressSchemaESPXNative = "espx_native"
)

func (c *Config) IsESPXNativeIngress() bool {
	if c == nil {
		return true
	}
	switch c.IngressSchema {
	case "", IngressSchemaESPXNative:
		return true
	default:
		return false
	}
}
