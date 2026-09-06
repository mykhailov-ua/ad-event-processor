package licensing

const (
	MCKFeatureBitOpenRTB uint8 = 0x01
)

func MCKFeatureBitsFromWork(mckWork [32]byte) uint8 {
	return mckWork[16]
}
