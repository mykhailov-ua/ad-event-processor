package domain

import "hash/crc32"

// GeoHashFromCountry maps ISO country code to RTB catalog shard key (CRC32-IEEE).
func GeoHashFromCountry(country string) uint32 {
	if country == "" {
		return 0
	}
	return crc32.ChecksumIEEE([]byte(country))
}

func GeoHashFromCountryBytes(country []byte) uint32 {
	if len(country) == 0 {
		return 0
	}
	return crc32.ChecksumIEEE(country)
}
