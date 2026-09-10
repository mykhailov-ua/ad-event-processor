package licensing

import entitlements "ad-event-processor/internal/licensing/entitlements"

const (
	MCKFeatureBitOpenRTB      = entitlements.MCKFeatureBitOpenRTB
	MCKFeatureBitMlFraudBoost = entitlements.MCKFeatureBitMlFraudBoost
	MCKFeatureBitEbpfEdge     = entitlements.MCKFeatureBitEbpfEdge
)

func MCKFeatureBitsFromWork(mckWork [32]byte) uint8 {
	return mckWork[16]
}

func MCKHasOpenRTB(bits uint8) bool {
	return bits&MCKFeatureBitOpenRTB != 0
}

func MCKHasMlFraudBoost(bits uint8) bool {
	return bits&MCKFeatureBitMlFraudBoost != 0
}

func MCKHasEbpfEdge(bits uint8) bool {
	return bits&MCKFeatureBitEbpfEdge != 0
}
