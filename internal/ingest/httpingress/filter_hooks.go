package httpingress

import "ad-event-processor/internal/filter"

func RegisterFilterHooks() {
	filter.HTTP1HeaderOrderMismatchFn = HTTP1HeaderOrderMismatch
	filter.H2SettingsAnomalyFn = H2SettingsAnomaly
	filter.H2PseudoOrderMismatchFn = H2PseudoOrderMismatch
	filter.H2DowngradeArtifactFn = H2DowngradeArtifact
}
