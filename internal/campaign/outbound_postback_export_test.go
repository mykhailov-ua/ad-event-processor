package campaign

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboundPostbackExport_roundTrip(t *testing.T) {
	t.Parallel()
	rows := []CampaignExportOutboundPostback{
		{
			Name:         "Webhook conversion",
			Priority:     10,
			Enabled:      true,
			Provider:     "webhook",
			URLTemplate:  "https://example.com/postback?cid={click_id}",
			TargetEvent:  "Purchase",
			TriggerKind:  OutboundTriggerConversion,
			TriggerValue: "",
		},
		{
			Name:         "Hold status",
			Priority:     20,
			Enabled:      true,
			Provider:     "webhook",
			URLTemplate:  "https://example.com/hold?status={status}",
			TriggerKind:  OutboundTriggerStatus,
			TriggerValue: "hold",
		},
	}
	writes, err := OutboundPostbackWritesFromExport(rows)
	require.NoError(t, err)
	require.Len(t, writes, 2)
	assert.Equal(t, "Webhook conversion", writes[0].Name)
	assert.Equal(t, OutboundTriggerStatus, writes[1].TriggerKind)
	assert.Equal(t, "hold", writes[1].TriggerValue)

	exported := ExportOutboundPostbacksFromRows(nil)
	assert.Empty(t, exported)
}
