package controlplane

import (
	"context"

	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"

	"github.com/google/uuid"
)

type commercialAdminAdapter struct {
	svc *Service
}

func (a commercialAdminAdapter) ListBrandsByCustomer(ctx context.Context, customerID uuid.UUID) ([]adminapi.BrandDTO, error) {
	rows, err := a.svc.ListBrandsByCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}
	out := make([]adminapi.BrandDTO, len(rows))
	for i, r := range rows {
		out[i] = adminapi.BrandDTO{
			ID:         r.ID,
			CustomerID: r.CustomerID,
			Name:       r.Name,
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  r.UpdatedAt,
			FreqLimit:  r.FreqLimit,
			FreqWindow: r.FreqWindow,
		}
	}
	return out, nil
}

func (a commercialAdminAdapter) CreateBrand(ctx context.Context, customerID uuid.UUID, name string) (uuid.UUID, error) {
	return a.svc.CreateBrand(ctx, customerID, name)
}

func (a commercialAdminAdapter) ListBrandCreatives(ctx context.Context, brandID uuid.UUID) ([]adminapi.BrandCreativeDTO, error) {
	rows, err := a.svc.ListBrandCreatives(ctx, brandID)
	if err != nil {
		return nil, err
	}
	out := make([]adminapi.BrandCreativeDTO, len(rows))
	for i, r := range rows {
		out[i] = adminapi.BrandCreativeDTO{
			ID:         r.ID,
			BrandID:    r.BrandID,
			Name:       r.Name,
			LandingURL: r.LandingURL,
			Weight:     r.Weight,
			Status:     r.Status,
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  r.UpdatedAt,
		}
	}
	return out, nil
}

func (a commercialAdminAdapter) UpsertBrandCreative(ctx context.Context, brandID uuid.UUID, name, landingURL string, weight int32, status string) (uuid.UUID, error) {
	return a.svc.UpsertBrandCreative(ctx, brandID, name, landingURL, weight, status)
}

func (a commercialAdminAdapter) UpdateBrandCreative(ctx context.Context, creativeID uuid.UUID, name, landingURL string, weight int32, status string) error {
	return a.svc.UpdateBrandCreative(ctx, creativeID, name, landingURL, weight, status)
}

func (a commercialAdminAdapter) DeleteBrandCreative(ctx context.Context, creativeID uuid.UUID) error {
	return a.svc.DeleteBrandCreative(ctx, creativeID)
}

func (a commercialAdminAdapter) ListSellers(ctx context.Context) ([]adminapi.SellerDTO, error) {
	rows, err := a.svc.ListSellers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]adminapi.SellerDTO, len(rows))
	for i, r := range rows {
		out[i] = mapSellerDTO(r)
	}
	return out, nil
}

func (a commercialAdminAdapter) CreateSeller(ctx context.Context, req adminapi.SellerWriteRequest) (adminapi.SellerDTO, error) {
	row, err := a.svc.CreateSeller(ctx, SellerCreateSpec{
		SellerID:       req.SellerID,
		Domain:         req.Domain,
		SellerType:     req.SellerType,
		Name:           req.Name,
		IsConfidential: req.IsConfidential,
	})
	if err != nil {
		return adminapi.SellerDTO{}, err
	}
	return mapSellerDTO(row), nil
}

func (a commercialAdminAdapter) UpdateSeller(ctx context.Context, id int64, req adminapi.SellerWriteRequest) (adminapi.SellerDTO, error) {
	row, err := a.svc.UpdateSeller(ctx, id, SellerUpdateSpec{
		SellerID:       req.SellerID,
		Domain:         req.Domain,
		SellerType:     req.SellerType,
		Name:           req.Name,
		IsConfidential: req.IsConfidential,
	})
	if err != nil {
		return adminapi.SellerDTO{}, err
	}
	return mapSellerDTO(row), nil
}

func (a commercialAdminAdapter) DeleteSeller(ctx context.Context, id int64) error {
	return a.svc.DeleteSeller(ctx, id)
}

func (a commercialAdminAdapter) ListAdsTxtEntries(ctx context.Context) ([]adminapi.AdsTxtEntryDTO, error) {
	rows, err := a.svc.ListAdsTxtEntries(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]adminapi.AdsTxtEntryDTO, len(rows))
	for i, r := range rows {
		out[i] = mapAdsTxtDTO(r)
	}
	return out, nil
}

func (a commercialAdminAdapter) CreateAdsTxtEntry(ctx context.Context, req adminapi.AdsTxtWriteRequest) (adminapi.AdsTxtEntryDTO, error) {
	row, err := a.svc.CreateAdsTxtEntry(ctx, AdsTxtEntryCreateSpec{
		Domain:             req.Domain,
		PublisherAccountID: req.PublisherAccountID,
		Relationship:       req.Relationship,
		CertAuthorityID:    req.CertAuthorityID,
		SortOrder:          req.SortOrder,
	})
	if err != nil {
		return adminapi.AdsTxtEntryDTO{}, err
	}
	return mapAdsTxtDTO(row), nil
}

func (a commercialAdminAdapter) UpdateAdsTxtEntry(ctx context.Context, id int64, req adminapi.AdsTxtWriteRequest) (adminapi.AdsTxtEntryDTO, error) {
	row, err := a.svc.UpdateAdsTxtEntry(ctx, id, AdsTxtEntryUpdateSpec{
		Domain:             req.Domain,
		PublisherAccountID: req.PublisherAccountID,
		Relationship:       req.Relationship,
		CertAuthorityID:    req.CertAuthorityID,
		SortOrder:          req.SortOrder,
	})
	if err != nil {
		return adminapi.AdsTxtEntryDTO{}, err
	}
	return mapAdsTxtDTO(row), nil
}

func (a commercialAdminAdapter) DeleteAdsTxtEntry(ctx context.Context, id int64) error {
	return a.svc.DeleteAdsTxtEntry(ctx, id)
}

func (a commercialAdminAdapter) BuildSellersJSON(ctx context.Context) ([]byte, error) {
	return a.svc.BuildSellersJSON(ctx)
}

func (a commercialAdminAdapter) BuildAdsTxt(ctx context.Context) (string, error) {
	return a.svc.BuildAdsTxt(ctx)
}

func (a commercialAdminAdapter) SupplyExportPath() string {
	return a.svc.SupplyExportPath()
}

func mapSellerDTO(r SellerDTO) adminapi.SellerDTO {
	return adminapi.SellerDTO{
		ID:             r.ID,
		SellerID:       r.SellerID,
		Domain:         r.Domain,
		SellerType:     r.SellerType,
		Name:           r.Name,
		IsConfidential: r.IsConfidential,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func mapAdsTxtDTO(r AdsTxtEntryDTO) adminapi.AdsTxtEntryDTO {
	return adminapi.AdsTxtEntryDTO{
		ID:                 r.ID,
		Domain:             r.Domain,
		PublisherAccountID: r.PublisherAccountID,
		Relationship:       r.Relationship,
		CertAuthorityID:    r.CertAuthorityID,
		SortOrder:          r.SortOrder,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}
}
