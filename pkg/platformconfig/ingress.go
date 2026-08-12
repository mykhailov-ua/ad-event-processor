package platformconfig

import (
	"log/slog"

	"github.com/bidshard/ad-event-processor/pkg/naming"
)

// NormalizeIngressSchema maps deprecated ingress values to the canonical schema.
func NormalizeIngressSchema(raw string) string {
	switch raw {
	case naming.DeprecatedIngressNativeSchema(), "native_v1":
		slog.Warn("deprecated ingress schema", "legacy", raw, "use", IngressAdEventProcessorNative)
		return IngressAdEventProcessorNative
	default:
		return raw
	}
}
