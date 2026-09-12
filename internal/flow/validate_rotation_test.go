package flow

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestValidatePathShape_rotationMode(t *testing.T) {
	t.Parallel()
	landerID := uuid.New()
	offerID := uuid.New()
	err := ValidatePathShape([]PathDTO{{
		Weight:       100,
		RotationMode: "unseen",
		Landers:      []PathLanderRef{{LanderID: landerID, Weight: 100}},
		Offers:       []PathOfferRef{{OfferID: offerID, Weight: 100}},
	}})
	require.NoError(t, err)

	err = ValidatePathShape([]PathDTO{{
		Weight:       100,
		RotationMode: "sequential",
		Landers:      []PathLanderRef{{LanderID: landerID, Weight: 100}},
		Offers:       []PathOfferRef{{OfferID: offerID, Weight: 100}},
	}})
	require.NoError(t, err)

	err = ValidatePathShape([]PathDTO{{
		Weight:       100,
		RotationMode: "bogus",
		Landers:      []PathLanderRef{{LanderID: landerID, Weight: 100}},
		Offers:       []PathOfferRef{{OfferID: offerID, Weight: 100}},
	}})
	require.Error(t, err)
}
