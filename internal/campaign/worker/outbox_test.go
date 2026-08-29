package worker

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDeliveryOutboxMerge_priority(t *testing.T) {
	t.Parallel()
	campID := uuid.New()
	merge := make(DeliveryOutboxMerge)

	merge.Upsert(campID, OutboxPriCreateCampaign, "CREATE_CAMPAIGN", []byte(`{"campaign_id":"x"}`))
	merge.Upsert(campID, OutboxPriPacing, "UPDATE_CAMPAIGN_PACING", []byte(`{"campaign_id":"x","pacing_mode":"EVEN"}`))
	merge.Upsert(campID, OutboxPriCreateCampaign, "CREATE_CAMPAIGN", []byte(`{"campaign_id":"x","budget":1}`))

	entry := merge[campID]
	assert.Equal(t, "UPDATE_CAMPAIGN_PACING", entry.EventType)
	assert.Equal(t, OutboxPriPacing, entry.Priority)
}
