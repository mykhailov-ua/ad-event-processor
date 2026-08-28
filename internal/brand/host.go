package brand

import (
	"context"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

type Host interface {
	ErrCustomerNotFound() error
	ErrBrandNotFound() error
	ErrCreativeNotFound() error
	ErrWeightMustBePositive() error
	ErrCreativeStatusInvalid() error
	MapNotFound(err, target error) error
	OnConfigureBrandFcap(ctx context.Context, q db.Querier, brandID uuid.UUID, prev db.AdvertiserBrand, limit, window int32) error
	OnBrandCreativesChanged(ctx context.Context, q db.Querier, brandID uuid.UUID) error
}
