package fraudadmin

import (
	"fmt"

	"ad-event-processor/pkg/piihash"
)

func HashIP(saltVersion uint8, saltHex, tokenKey, ip string) ([16]byte, error) {
	hasher, err := piihash.NewFromSalt(saltVersion, saltHex, tokenKey)
	if err != nil {
		return [16]byte{}, fmt.Errorf("piihash: %w", err)
	}
	return hasher.HashIP(ip), nil
}
