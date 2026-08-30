package trafficoptimizer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func ApplyActionHash(ruleID uuid.UUID, flowID uuid.UUID, windowEnd time.Time) string {
	raw := fmt.Sprintf("%s|%s|%s|apply_weights", ruleID, flowID, windowEnd.UTC().Format(time.RFC3339))
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
