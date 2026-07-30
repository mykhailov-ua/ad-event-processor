package partition

import "hash/crc32"

var castagnoli = crc32.MakeTable(crc32.Castagnoli)

const slotMask = 1023

func Slot(campaignID []byte) uint32 {
	if len(campaignID) < 16 {
		return 0
	}
	return crc32.Checksum(campaignID[:16], castagnoli) & slotMask
}

func Index(campaignID []byte, numPartitions int) uint16 {
	if numPartitions <= 1 {
		return 0
	}
	return uint16(int(Slot(campaignID)) % numPartitions)
}
