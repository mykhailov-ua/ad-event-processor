package ingestion

func BoostPPMFromUint8(boost uint8) uint32 {
	if boost == 0 {
		return CTRPPMUnit
	}
	return CTRPPMUnit + uint32(boost)*10_000
}
