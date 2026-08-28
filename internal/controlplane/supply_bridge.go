package controlplane

import (
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/supply"
	"ad-event-processor/pkg/coldpath"
	"context"

	"github.com/google/uuid"
)

var _ supply.Host = (*Service)(nil)

func (s *Service) SupplyStore() *supply.Store {
	if s == nil {
		return nil
	}
	if s.supplyStore == nil {
		s.supplyStore = supply.NewStore(s.pool, s)
	}
	return s.supplyStore
}

func (s *Service) ActorUserID(ctx context.Context) uuid.UUID {
	if u, ok := GetUser(ctx); ok {
		return u.UserID
	}
	return uuid.Nil
}

func (s *Service) ErrValidation(msg string) error {
	return errValidation(msg)
}

func (s *Service) EnqueueSupplyFilesUpdate(ctx context.Context, q db.Querier, trigger string) error {
	payload, err := coldpath.MarshalOutbox(supply.FilesPayload{Trigger: trigger})
	if err != nil {
		return err
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "UPDATE_SUPPLY_FILES",
		Payload:   payload,
	})
	return err
}

func (s *Service) SupplyExportPath() string {
	if s.cfg != nil && s.cfg.Management.SupplyExportPath != "" {
		return s.cfg.Management.SupplyExportPath
	}
	return "./data/supply-export"
}

func (s *Service) ListSellers(ctx context.Context) ([]supply.SellerDTO, error) {
	return s.SupplyStore().ListSellers(ctx)
}

func (s *Service) GetSeller(ctx context.Context, id int64) (supply.SellerDTO, error) {
	return s.SupplyStore().GetSeller(ctx, id)
}

func (s *Service) CreateSeller(ctx context.Context, spec supply.SellerCreateSpec) (supply.SellerDTO, error) {
	return s.SupplyStore().CreateSeller(ctx, spec)
}

func (s *Service) UpdateSeller(ctx context.Context, id int64, spec supply.SellerUpdateSpec) (supply.SellerDTO, error) {
	return s.SupplyStore().UpdateSeller(ctx, id, spec)
}

func (s *Service) DeleteSeller(ctx context.Context, id int64) error {
	return s.SupplyStore().DeleteSeller(ctx, id)
}

func (s *Service) ListAdsTxtEntries(ctx context.Context) ([]supply.AdsTxtEntryDTO, error) {
	return s.SupplyStore().ListAdsTxtEntries(ctx)
}

func (s *Service) GetAdsTxtEntry(ctx context.Context, id int64) (supply.AdsTxtEntryDTO, error) {
	return s.SupplyStore().GetAdsTxtEntry(ctx, id)
}

func (s *Service) CreateAdsTxtEntry(ctx context.Context, spec supply.AdsTxtEntryCreateSpec) (supply.AdsTxtEntryDTO, error) {
	return s.SupplyStore().CreateAdsTxtEntry(ctx, spec)
}

func (s *Service) UpdateAdsTxtEntry(ctx context.Context, id int64, spec supply.AdsTxtEntryUpdateSpec) (supply.AdsTxtEntryDTO, error) {
	return s.SupplyStore().UpdateAdsTxtEntry(ctx, id, spec)
}

func (s *Service) DeleteAdsTxtEntry(ctx context.Context, id int64) error {
	return s.SupplyStore().DeleteAdsTxtEntry(ctx, id)
}

func (s *Service) BuildSellersJSON(ctx context.Context) ([]byte, error) {
	return s.SupplyStore().BuildSellersJSON(ctx)
}

func (s *Service) GetSellersJSON(ctx context.Context) ([]byte, error) {
	return s.SupplyStore().GetSellersJSON(ctx)
}

func (s *Service) BuildAdsTxt(ctx context.Context) (string, error) {
	return s.SupplyStore().BuildAdsTxt(ctx)
}

type supplyAdminHost struct {
	svc *Service
}

var _ supply.AdminHost = supplyAdminHost{}

func (h supplyAdminHost) ListSellers(ctx context.Context) ([]supply.SellerDTO, error) {
	return h.svc.ListSellers(ctx)
}

func (h supplyAdminHost) CreateSeller(ctx context.Context, spec supply.SellerCreateSpec) (supply.SellerDTO, error) {
	return h.svc.CreateSeller(ctx, spec)
}

func (h supplyAdminHost) UpdateSeller(ctx context.Context, id int64, spec supply.SellerUpdateSpec) (supply.SellerDTO, error) {
	return h.svc.UpdateSeller(ctx, id, spec)
}

func (h supplyAdminHost) DeleteSeller(ctx context.Context, id int64) error {
	return h.svc.DeleteSeller(ctx, id)
}

func (h supplyAdminHost) ListAdsTxtEntries(ctx context.Context) ([]supply.AdsTxtEntryDTO, error) {
	return h.svc.ListAdsTxtEntries(ctx)
}

func (h supplyAdminHost) CreateAdsTxtEntry(ctx context.Context, spec supply.AdsTxtEntryCreateSpec) (supply.AdsTxtEntryDTO, error) {
	return h.svc.CreateAdsTxtEntry(ctx, spec)
}

func (h supplyAdminHost) UpdateAdsTxtEntry(ctx context.Context, id int64, spec supply.AdsTxtEntryUpdateSpec) (supply.AdsTxtEntryDTO, error) {
	return h.svc.UpdateAdsTxtEntry(ctx, id, spec)
}

func (h supplyAdminHost) DeleteAdsTxtEntry(ctx context.Context, id int64) error {
	return h.svc.DeleteAdsTxtEntry(ctx, id)
}

func (h supplyAdminHost) BuildSellersJSON(ctx context.Context) ([]byte, error) {
	return h.svc.BuildSellersJSON(ctx)
}

func (h supplyAdminHost) BuildAdsTxt(ctx context.Context) (string, error) {
	return h.svc.BuildAdsTxt(ctx)
}

func (h supplyAdminHost) SupplyExportPath() string {
	return h.svc.SupplyExportPath()
}

func (h supplyAdminHost) ValidateSupplyFiles(ctx context.Context) (supply.ValidationDTO, error) {
	report, err := h.svc.ValidateSupplyFiles(ctx)
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
