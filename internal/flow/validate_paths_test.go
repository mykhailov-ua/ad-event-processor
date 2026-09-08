package flow_test

import (
	"context"
	"testing"

	"ad-event-processor/internal/flow"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestValidatePathWeightSum_holdout(t *testing.T) {
	t.Parallel()
	landerID := uuid.New()
	offerID := uuid.New()
	paths := []flow.PathDTO{
		{
			Weight:  50,
			Landers: []flow.PathLanderRef{{LanderID: landerID, Weight: 100}},
			Offers:  []flow.PathOfferRef{{OfferID: offerID, Weight: 100}},
		},
		{
			Weight:  49,
			Landers: []flow.PathLanderRef{{LanderID: landerID, Weight: 100}},
			Offers:  []flow.PathOfferRef{{OfferID: offerID, Weight: 100}},
		},
	}
	require.Error(t, flow.ValidatePathWeightSum(paths))
	require.Error(t, flow.ValidatePaths(paths))
}

func TestValidatePaths_validSum(t *testing.T) {
	t.Parallel()
	landerID := uuid.New()
	offerID := uuid.New()
	paths := []flow.PathDTO{{
		Weight:  100,
		Landers: []flow.PathLanderRef{{LanderID: landerID, Weight: 100}},
		Offers:  []flow.PathOfferRef{{OfferID: offerID, Weight: 100}},
	}}
	require.NoError(t, flow.ValidatePaths(paths))
}

func TestValidatePathRefs_weightSum_holdout(t *testing.T) {
	t.Parallel()
	landerID := uuid.New()
	offerID := uuid.New()
	paths := []flow.PathDTO{{
		Weight:  90,
		Landers: []flow.PathLanderRef{{LanderID: landerID, Weight: 100}},
		Offers:  []flow.PathOfferRef{{OfferID: offerID, Weight: 100}},
	}}
	err := flow.ValidatePathRefs(context.Background(), nil, paths)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sum to 100")
}
