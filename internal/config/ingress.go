package config

import (
	"log/slog"
	"sync"
)

const (
	IngressSchemaOpenRTB3               = "openrtb_3"
	IngressSchemaAdEventProcessorNative = "ad_event_processor_native"

	IngressAdEventProcessorNative = IngressSchemaAdEventProcessorNative

	IngressSchemaNativeV1 = "native_v1"
)

var ingressLegacyWarnOnce sync.Once

func SupportedIngressSchemas() []string {
	return []string{IngressSchemaOpenRTB3, IngressSchemaAdEventProcessorNative}
}

func NormalizeIngressSchema(raw string) string {
	switch raw {
	case LegacyIngressNativeSchema(), IngressSchemaNativeV1:
		ingressLegacyWarnOnce.Do(func() {
			slog.Warn("deprecated ingress schema", "legacy", raw, "use", IngressSchemaAdEventProcessorNative)
		})
		return IngressSchemaAdEventProcessorNative
	default:
		return raw
	}
}

func (c *Config) IsAdEventProcessorNativeIngress() bool {
	if c == nil {
		return true
	}
	switch c.IngressSchema {
	case "", IngressSchemaAdEventProcessorNative, LegacyIngressNativeSchema(), IngressSchemaNativeV1:
		return true
	default:
		return false
	}
}

func (c *Config) IsLegacyNativeIngress() bool {
	return c.IsAdEventProcessorNativeIngress()
}
