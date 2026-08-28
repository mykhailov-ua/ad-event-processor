package controlplane

import (
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/flow"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFlowPathShape_holdout(t *testing.T) {
	t.Parallel()
	landerID := uuid.New()
	offerID := uuid.New()
	valid := []campaign.FlowPathDTO{{
		Weight: 100,
		Landers: []campaign.FlowPathLanderRef{{
			LanderID: landerID,
			Weight:   50,
		}},
		Offers: []campaign.FlowPathOfferRef{{
			OfferID: offerID,
			Weight:  100,
		}},
	}}
	require.NoError(t, flow.ValidatePathShape(valid))

	bad := valid
	bad[0].Landers[0].Weight = 0
	assert.Error(t, flow.ValidatePathShape(bad))

	empty := []campaign.FlowPathDTO{}
	assert.Error(t, flow.ValidatePathShape(empty))
}
