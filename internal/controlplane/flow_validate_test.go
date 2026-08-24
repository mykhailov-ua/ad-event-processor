package controlplane

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFlowPathShape_holdout(t *testing.T) {
	t.Parallel()
	landerID := uuid.New()
	offerID := uuid.New()
	valid := []FlowPathDTO{{
		Weight: 100,
		Landers: []FlowPathLanderRef{{
			LanderID: landerID,
			Weight:   50,
		}},
		Offers: []FlowPathOfferRef{{
			OfferID: offerID,
			Weight:  100,
		}},
	}}
	require.NoError(t, validateFlowPathShape(valid))

	bad := valid
	bad[0].Landers[0].Weight = 0
	assert.Error(t, validateFlowPathShape(bad))

	empty := []FlowPathDTO{}
	assert.Error(t, validateFlowPathShape(empty))
}
