package stream

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamConsumer_parseFraudAggregate(t *testing.T) {
	consumer := &StreamConsumer{}
	evt := consumer.parseMessage("1-0", map[string]interface{}{
		"type":         "fraud_aggregate",
		"subnet":       "10.0.0.0/24",
		"ipv6_prefix":  "2001:db8::/64",
		"fraud_reason": "low_ttc",
		"count":        "1500",
		"window_ms":    "75",
	})
	require.NotNil(t, evt)
	assert.Equal(t, fraudAggregateEventType, evt.Type)
	assert.Equal(t, "10.0.0.0/24", evt.IP)
	assert.Equal(t, "2001:db8::/64", evt.PlacementID)
	assert.Equal(t, "low_ttc", evt.FraudReason)
	assert.Equal(t, "1500", evt.ClickID)
	assert.Equal(t, "75", evt.UserID)
	count, window := fraudAggregateFields(evt)
	assert.Equal(t, uint64(1500), count)
	assert.Equal(t, uint32(75), window)
}
