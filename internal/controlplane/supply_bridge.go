package controlplane

import (
	"context"

	"ad-event-processor/internal/supply"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

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
	payload, err := coldpath.MarshalOutbox(SupplyFilesPayload{Trigger: trigger})
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

func (s *Service) ListSellers(ctx context.Context) ([]SellerDTO, error) {
	return s.SupplyStore().ListSellers(ctx)
}

func (s *Service) GetSeller(ctx context.Context, id int64) (SellerDTO, error) {
	return s.SupplyStore().GetSeller(ctx, id)
}

func (s *Service) CreateSeller(ctx context.Context, spec SellerCreateSpec) (SellerDTO, error) {
	return s.SupplyStore().CreateSeller(ctx, spec)
}

func (s *Service) UpdateSeller(ctx context.Context, id int64, spec SellerUpdateSpec) (SellerDTO, error) {
	return s.SupplyStore().UpdateSeller(ctx, id, spec)
}

func (s *Service) DeleteSeller(ctx context.Context, id int64) error {
	return s.SupplyStore().DeleteSeller(ctx, id)
}

func (s *Service) ListAdsTxtEntries(ctx context.Context) ([]AdsTxtEntryDTO, error) {
	return s.SupplyStore().ListAdsTxtEntries(ctx)
}

func (s *Service) GetAdsTxtEntry(ctx context.Context, id int64) (AdsTxtEntryDTO, error) {
	return s.SupplyStore().GetAdsTxtEntry(ctx, id)
}

func (s *Service) CreateAdsTxtEntry(ctx context.Context, spec AdsTxtEntryCreateSpec) (AdsTxtEntryDTO, error) {
	return s.SupplyStore().CreateAdsTxtEntry(ctx, spec)
}

func (s *Service) UpdateAdsTxtEntry(ctx context.Context, id int64, spec AdsTxtEntryUpdateSpec) (AdsTxtEntryDTO, error) {
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
