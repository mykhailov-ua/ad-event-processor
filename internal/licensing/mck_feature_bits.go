package licensing

const (
	MCKFeatureBitOpenRTB uint8 = 0x01
)

func MCKFeatureBitsFromWork(mckWork [32]byte) uint8 {
	return mckWork[16]
}

func mckFeatureBitOpenRTBSet(bits uint8) bool {
	return bits&MCKFeatureBitOpenRTB != 0
}
