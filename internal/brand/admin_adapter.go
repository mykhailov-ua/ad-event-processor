package brand

import (
	"context"

	"github.com/google/uuid"
)

type adminAdapter struct {
	svc AdminService
}

func NewAdminAdapter(svc AdminService) AdminService {
	if svc == nil {
		return nil
	}
	return adminAdapter{svc: svc}
}

func (a adminAdapter) ListBrandsByCustomer(ctx context.Context, customerID uuid.UUID) ([]DTO, error) {
	return a.svc.ListBrandsByCustomer(ctx, customerID)
}

func (a adminAdapter) CreateBrand(ctx context.Context, customerID uuid.UUID, name string) (uuid.UUID, error) {
	return a.svc.CreateBrand(ctx, customerID, name)
}

func (a adminAdapter) ListBrandCreatives(ctx context.Context, brandID uuid.UUID) ([]CreativeDTO, error) {
	return a.svc.ListBrandCreatives(ctx, brandID)
}

func (a adminAdapter) UpsertBrandCreative(ctx context.Context, brandID uuid.UUID, name, landingURL string, weight int32, status string) (uuid.UUID, error) {
	return a.svc.UpsertBrandCreative(ctx, brandID, name, landingURL, weight, status)
}

func (a adminAdapter) UpdateBrandCreative(ctx context.Context, creativeID uuid.UUID, name, landingURL string, weight int32, status string) error {
	return a.svc.UpdateBrandCreative(ctx, creativeID, name, landingURL, weight, status)
}

func (a adminAdapter) DeleteBrandCreative(ctx context.Context, creativeID uuid.UUID) error {
	return a.svc.DeleteBrandCreative(ctx, creativeID)
}
