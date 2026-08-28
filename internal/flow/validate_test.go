package flow_test

import (
	"testing"

	"ad-event-processor/internal/flow"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestValidatePathShape_filters_holdout(t *testing.T) {
	t.Parallel()
	require.NoError(t, flow.ValidatePathShape([]flow.PathDTO{{
		Weight:  100,
		Landers: []flow.PathLanderRef{{LanderID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Weight: 1}},
		Offers:  []flow.PathOfferRef{{OfferID: uuid.MustParse("00000000-0000-4000-8000-000000000002"), Weight: 1}},
		Filters: &flow.PathFiltersDTO{
			Countries: []string{"us"},
			Devices:   []string{"mobile"},
			OS:        []string{"ios"},
			Languages: []string{"en"},
		},
	}}))
}

func TestBuildValidateResponse_weightSum_holdout(t *testing.T) {
	t.Parallel()
	resp := flow.BuildValidateResponse([]flow.PathDTO{
		{Weight: 50, Landers: []flow.PathLanderRef{{LanderID: uuid.New(), Weight: 1}}, Offers: []flow.PathOfferRef{{OfferID: uuid.New(), Weight: 1}}},
		{Weight: 49, Landers: []flow.PathLanderRef{{LanderID: uuid.New(), Weight: 1}}, Offers: []flow.PathOfferRef{{OfferID: uuid.New(), Weight: 1}}},
	})
	require.False(t, resp.Valid)
	require.Equal(t, "weight_sum", resp.PathErrors[0].Code)
}
