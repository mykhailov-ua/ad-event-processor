package entitlements

const (
	MCKFeatureBitOpenRTB      uint8 = 0x01
	MCKFeatureBitMlFraudBoost uint8 = 0x02
	MCKFeatureBitEbpfEdge     uint8 = 0x04
)

func MCKFeatureBitOpenRTBVal() uint8 {
	return MCKFeatureBitOpenRTB
}

func MCKFeatureBitMlFraudBoostVal() uint8 {
	return MCKFeatureBitMlFraudBoost
}

func MCKFeatureBitEbpfEdgeVal() uint8 {
	return MCKFeatureBitEbpfEdge
}
