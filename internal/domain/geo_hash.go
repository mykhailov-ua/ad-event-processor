package domain

import "hash/crc32"

// GeoHashFromCountry maps ISO country code to RTB catalog shard key (CRC32-IEEE).
func GeoHashFromCountry(country string) uint32 {
	if country == "" {
		return 0
	}
	crc := uint32(0xffffffff)
	for i := 0; i < len(country); i++ {
		crc = crc32.IEEETable[byte(crc^uint32(country[i]))] ^ (crc >> 8)
	}
	return crc ^ 0xffffffff
}

func GeoHashFromCountryBytes(country []byte) uint32 {
	if len(country) == 0 {
		return 0
	}
	return crc32.ChecksumIEEE(country)
}
