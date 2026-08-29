package controlplane

import (
	"testing"

	"ad-event-processor/internal/flow"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFlowPathShape_holdout(t *testing.T) {
	t.Parallel()
	landerID := uuid.New()
	offerID := uuid.New()
	valid := []flow.PathDTO{{
		Weight: 100,
		Landers: []flow.PathLanderRef{{
			LanderID: landerID,
			Weight:   50,
		}},
		Offers: []flow.PathOfferRef{{
			OfferID: offerID,
			Weight:  100,
		}},
	}}
	require.NoError(t, flow.ValidatePathShape(valid))

	bad := valid
	bad[0].Landers[0].Weight = 0
	assert.Error(t, flow.ValidatePathShape(bad))

	empty := []flow.PathDTO{}
	assert.Error(t, flow.ValidatePathShape(empty))
}
