package supply

import (
	"context"
)

type AdminHost interface {
	ListSellers(ctx context.Context) ([]SellerDTO, error)
	CreateSeller(ctx context.Context, spec SellerCreateSpec) (SellerDTO, error)
	UpdateSeller(ctx context.Context, id int64, spec SellerUpdateSpec) (SellerDTO, error)
	DeleteSeller(ctx context.Context, id int64) error
	ListAdsTxtEntries(ctx context.Context) ([]AdsTxtEntryDTO, error)
	CreateAdsTxtEntry(ctx context.Context, spec AdsTxtEntryCreateSpec) (AdsTxtEntryDTO, error)
	UpdateAdsTxtEntry(ctx context.Context, id int64, spec AdsTxtEntryUpdateSpec) (AdsTxtEntryDTO, error)
	DeleteAdsTxtEntry(ctx context.Context, id int64) error
	BuildSellersJSON(ctx context.Context) ([]byte, error)
	BuildAdsTxt(ctx context.Context) (string, error)
	SupplyExportPath() string
	ValidateSupplyFiles(ctx context.Context) (ValidationDTO, error)
}

type adminAdapter struct {
	host AdminHost
}

func NewAdminAdapter(host AdminHost) AdminService {
	if host == nil {
		return nil
	}
	return adminAdapter{host: host}
}

func (a adminAdapter) ListSellers(ctx context.Context) ([]SellerDTO, error) {
	return a.host.ListSellers(ctx)
}

func (a adminAdapter) CreateSeller(ctx context.Context, req SellerWriteRequest) (SellerDTO, error) {
	return a.host.CreateSeller(ctx, SellerCreateSpec(req))
}

func (a adminAdapter) UpdateSeller(ctx context.Context, id int64, req SellerWriteRequest) (SellerDTO, error) {
	return a.host.UpdateSeller(ctx, id, SellerUpdateSpec(req))
}

func (a adminAdapter) DeleteSeller(ctx context.Context, id int64) error {
	return a.host.DeleteSeller(ctx, id)
}

func (a adminAdapter) ListAdsTxtEntries(ctx context.Context) ([]AdsTxtEntryDTO, error) {
	return a.host.ListAdsTxtEntries(ctx)
}

func (a adminAdapter) CreateAdsTxtEntry(ctx context.Context, req AdsTxtWriteRequest) (AdsTxtEntryDTO, error) {
	return a.host.CreateAdsTxtEntry(ctx, AdsTxtEntryCreateSpec(req))
}

func (a adminAdapter) UpdateAdsTxtEntry(ctx context.Context, id int64, req AdsTxtWriteRequest) (AdsTxtEntryDTO, error) {
	return a.host.UpdateAdsTxtEntry(ctx, id, AdsTxtEntryUpdateSpec(req))
}

func (a adminAdapter) DeleteAdsTxtEntry(ctx context.Context, id int64) error {
	return a.host.DeleteAdsTxtEntry(ctx, id)
}

func (a adminAdapter) BuildSellersJSON(ctx context.Context) ([]byte, error) {
	return a.host.BuildSellersJSON(ctx)
}

func (a adminAdapter) BuildAdsTxt(ctx context.Context) (string, error) {
	return a.host.BuildAdsTxt(ctx)
}

func (a adminAdapter) SupplyExportPath() string {
	return a.host.SupplyExportPath()
}

func (a adminAdapter) ValidateSupplyFiles(ctx context.Context) (ValidationDTO, error) {
	return a.host.ValidateSupplyFiles(ctx)
}
