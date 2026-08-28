package flow

import (
	"context"

	"github.com/google/uuid"
)

type PathRefChecker interface {
	ValidateLanderIDs(ctx context.Context, ids []uuid.UUID) error
	ValidateOfferIDs(ctx context.Context, ids []uuid.UUID) error
}

func ValidatePathRefs(ctx context.Context, checker PathRefChecker, paths []PathDTO) error {
	if err := ValidatePathShape(paths); err != nil {
		return err
	}
	if checker == nil {
		return nil
	}
	landerIDs := make([]uuid.UUID, 0)
	offerIDs := make([]uuid.UUID, 0)
	landerSet := make(map[uuid.UUID]struct{})
	offerSet := make(map[uuid.UUID]struct{})
	for _, path := range paths {
		for _, lander := range path.Landers {
			if _, ok := landerSet[lander.LanderID]; !ok {
				landerSet[lander.LanderID] = struct{}{}
				landerIDs = append(landerIDs, lander.LanderID)
			}
		}
		for _, offer := range path.Offers {
			if _, ok := offerSet[offer.OfferID]; !ok {
				offerSet[offer.OfferID] = struct{}{}
				offerIDs = append(offerIDs, offer.OfferID)
			}
		}
	}
	if err := checker.ValidateLanderIDs(ctx, landerIDs); err != nil {
		return err
	}
	return checker.ValidateOfferIDs(ctx, offerIDs)
}
