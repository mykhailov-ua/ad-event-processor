package licensing

const (
	// MCKFeatureBitOpenRTB requires stretched MCK work byte 16 bit 0 for OpenRTB when seed coupling is on.
	MCKFeatureBitOpenRTB uint8 = 0x01
)

// MCKFeatureBitsFromWork returns the stretched MCK feature byte used with JWT feature flags.
func MCKFeatureBitsFromWork(mckWork [32]byte) uint8 {
	return mckWork[16]
}

func mckFeatureBitOpenRTBSet(bits uint8) bool {
	return bits&MCKFeatureBitOpenRTB != 0
}
