package automation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func ActionHash(ruleID uuid.UUID, campaignID uuid.UUID, placementID string, windowEnd time.Time, actionType string) string {
	raw := fmt.Sprintf("%s|%s|%s|%s|%s", ruleID, campaignID, placementID, windowEnd.UTC().Format(time.RFC3339), actionType)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
