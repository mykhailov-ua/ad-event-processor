package ingestion

import (
	"crypto/sha256"
	"encoding/hex"
)

const (
	ConsentPurposeAdStorage     int16 = 1 << 0
	ConsentPurposeAnalytics     int16 = 1 << 1
	ConsentRedisKeyPrefix             = "consent:user:"
	ConsentDefaultUpdateChannel       = "consent:update"
)

func HashUserID(userID string) []byte {
	sum := sha256.Sum256([]byte(userID))
	return sum[:]
}

func HashUserIDHex(userID string) string {
	return hex.EncodeToString(HashUserID(userID))
}

func ConsentFlagsFromPurposes(purposes int16) (adStorage, analyticsStorage bool) {
	adStorage = purposes&ConsentPurposeAdStorage != 0
	analyticsStorage = purposes&ConsentPurposeAnalytics != 0
	return adStorage, analyticsStorage
}
