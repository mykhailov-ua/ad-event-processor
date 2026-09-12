package campaign

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestApplyFlowPathWeights_updatesAndValidates_holdout(t *testing.T) {
	t.Parallel()
	landerID := uuid.New()
	offerID := uuid.New()
	paths := []FlowPathDTO{{
		Weight:  60,
		Landers: []FlowPathLanderRef{{LanderID: landerID, Weight: 100}},
		Offers:  []FlowPathOfferRef{{OfferID: offerID, Weight: 100}},
	}, {
		Weight:  40,
		Landers: []FlowPathLanderRef{{LanderID: landerID, Weight: 100}},
		Offers:  []FlowPathOfferRef{{OfferID: offerID, Weight: 100}},
	}}
	updated, err := ApplyFlowPathWeights(paths, []BulkFlowPathWeight{
		{PathIndex: 0, Weight: 70},
		{PathIndex: 1, Weight: 30},
	})
	require.NoError(t, err)
	require.Equal(t, int32(70), updated[0].Weight)
	require.Equal(t, int32(30), updated[1].Weight)
}

func TestApplyFlowPathWeights_rejectsOutOfRange_holdout(t *testing.T) {
	t.Parallel()
	_, err := ApplyFlowPathWeights([]FlowPathDTO{{Weight: 100}}, []BulkFlowPathWeight{{PathIndex: 1, Weight: 10}})
	require.Error(t, err)
}
