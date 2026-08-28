package controlplane

import (
	"context"

	"ad-event-processor/internal/brand"

	"github.com/google/uuid"
)

type brandAdminAdapter struct {
	svc *Service
}

func (a brandAdminAdapter) ListBrandsByCustomer(ctx context.Context, customerID uuid.UUID) ([]brand.DTO, error) {
	return a.svc.ListBrandsByCustomer(ctx, customerID)
}

func (a brandAdminAdapter) CreateBrand(ctx context.Context, customerID uuid.UUID, name string) (uuid.UUID, error) {
	return a.svc.CreateBrand(ctx, customerID, name)
}

func (a brandAdminAdapter) ListBrandCreatives(ctx context.Context, brandID uuid.UUID) ([]brand.CreativeDTO, error) {
	return a.svc.ListBrandCreatives(ctx, brandID)
}

func (a brandAdminAdapter) UpsertBrandCreative(ctx context.Context, brandID uuid.UUID, name, landingURL string, weight int32, status string) (uuid.UUID, error) {
	return a.svc.UpsertBrandCreative(ctx, brandID, name, landingURL, weight, status)
}

func (a brandAdminAdapter) UpdateBrandCreative(ctx context.Context, creativeID uuid.UUID, name, landingURL string, weight int32, status string) error {
	return a.svc.UpdateBrandCreative(ctx, creativeID, name, landingURL, weight, status)
}

func (a brandAdminAdapter) DeleteBrandCreative(ctx context.Context, creativeID uuid.UUID) error {
	return a.svc.DeleteBrandCreative(ctx, creativeID)
}
