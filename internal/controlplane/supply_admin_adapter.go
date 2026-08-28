package controlplane

import (
	"context"

	"ad-event-processor/internal/supply"
)

type supplyAdminAdapter struct {
	svc *Service
}

func (a supplyAdminAdapter) ListSellers(ctx context.Context) ([]supply.SellerDTO, error) {
	return a.svc.ListSellers(ctx)
}

func (a supplyAdminAdapter) CreateSeller(ctx context.Context, req supply.SellerWriteRequest) (supply.SellerDTO, error) {
	return a.svc.CreateSeller(ctx, SellerCreateSpec(req))
}

func (a supplyAdminAdapter) UpdateSeller(ctx context.Context, id int64, req supply.SellerWriteRequest) (supply.SellerDTO, error) {
	return a.svc.UpdateSeller(ctx, id, SellerUpdateSpec(req))
}

func (a supplyAdminAdapter) DeleteSeller(ctx context.Context, id int64) error {
	return a.svc.DeleteSeller(ctx, id)
}

func (a supplyAdminAdapter) ListAdsTxtEntries(ctx context.Context) ([]supply.AdsTxtEntryDTO, error) {
	return a.svc.ListAdsTxtEntries(ctx)
}

func (a supplyAdminAdapter) CreateAdsTxtEntry(ctx context.Context, req supply.AdsTxtWriteRequest) (supply.AdsTxtEntryDTO, error) {
	return a.svc.CreateAdsTxtEntry(ctx, AdsTxtEntryCreateSpec(req))
}

func (a supplyAdminAdapter) UpdateAdsTxtEntry(ctx context.Context, id int64, req supply.AdsTxtWriteRequest) (supply.AdsTxtEntryDTO, error) {
	return a.svc.UpdateAdsTxtEntry(ctx, id, AdsTxtEntryUpdateSpec(req))
}

func (a supplyAdminAdapter) DeleteAdsTxtEntry(ctx context.Context, id int64) error {
	return a.svc.DeleteAdsTxtEntry(ctx, id)
}

func (a supplyAdminAdapter) BuildSellersJSON(ctx context.Context) ([]byte, error) {
	return a.svc.BuildSellersJSON(ctx)
}

func (a supplyAdminAdapter) BuildAdsTxt(ctx context.Context) (string, error) {
	return a.svc.BuildAdsTxt(ctx)
}

func (a supplyAdminAdapter) SupplyExportPath() string {
	return a.svc.SupplyExportPath()
}

func (a supplyAdminAdapter) ValidateSupplyFiles(ctx context.Context) (supply.ValidationDTO, error) {
	report, err := a.svc.ValidateSupplyFiles(ctx)
	if err != nil {
		return supply.ValidationDTO{}, err
	}
	return supply.ValidationDTO{
		SellersJSONValid:      report.SellersJSONValid,
		SellersChecksumSHA256: report.SellersChecksumSHA256,
		SellersCount:          report.SellersCount,
		AdsTxtValid:           report.AdsTxtValid,
		AdsTxtChecksumSHA256:  report.AdsTxtChecksumSHA256,
		AdsTxtLineCount:       report.AdsTxtLineCount,
		Issues:                report.Issues,
	}, nil
}
