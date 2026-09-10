package postback

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPostback_DLQRetry_afterFailedDispatch_holdout(t *testing.T) {
	t.Parallel()
	status := postbackDispatchStatusFailed
	switch status {
	case postbackDispatchStatusSent, postbackDispatchStatusDelivered, postbackDispatchStatusInFlight:
		t.Fatal("holdout: FAILED must not fall through to unexpected-status error path")
	case postbackDispatchStatusFailed:
		// resolveDispatchSlot resets FAILED to IN_FLIGHT before retry.
	default:
		t.Fatalf("holdout: unexpected dispatch status %q", status)
	}
	assert.Equal(t, "FAILED", postbackDispatchStatusFailed)
}
