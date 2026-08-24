package platformconfig

import (
	"log/slog"

	"ad-event-processor/pkg/naming"
)

func NormalizeIngressSchema(raw string) string {
	switch raw {
	case naming.DeprecatedIngressNativeSchema(), "native_v1":
		slog.Warn("deprecated ingress schema", "legacy", raw, "use", IngressAdEventProcessorNative)
		return IngressAdEventProcessorNative
	default:
		return raw
	}
}
