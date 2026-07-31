package config

import (
	"log/slog"
	"sync"
)

const (
	IngressSchemaOpenRTB3   = "openrtb_3"
	IngressSchemaNativeV1   = "native_v1"
	IngressSchemaESPXNative = "espx_native"
)

var ingressLegacyWarnOnce sync.Once

func SupportedIngressSchemas() []string {
	return []string{IngressSchemaOpenRTB3, IngressSchemaNativeV1}
}

func NormalizeIngressSchema(raw string) string {
	switch raw {
	case IngressSchemaESPXNative:
		ingressLegacyWarnOnce.Do(func() {
			slog.Warn("deprecated ingress schema", "legacy", IngressSchemaESPXNative, "use", IngressSchemaNativeV1)
		})
		return IngressSchemaNativeV1
	default:
		return raw
	}
}

func (c *Config) IsESPXNativeIngress() bool {
	if c == nil {
		return true
	}
	switch c.IngressSchema {
	case "", IngressSchemaESPXNative, IngressSchemaNativeV1:
		return true
	default:
		return false
	}
}
