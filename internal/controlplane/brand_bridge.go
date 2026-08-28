package controlplane

import (
	"context"
	"fmt"

	"ad-event-processor/internal/brand"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
)

var _ brand.Host = (*Service)(nil)

func (s *Service) BrandStore() *brand.Store {
	if s == nil {
		return nil
	}
	if s.brandStore == nil {
		s.brandStore = brand.NewStore(s.pool, s)
	}
	return s.brandStore
}

func (s *Service) ErrCustomerNotFound() error   { return ErrCustomerNotFound }
func (s *Service) ErrBrandNotFound() error      { return ErrBrandNotFound }
func (s *Service) ErrCreativeNotFound() error   { return ErrCreativeNotFound }
func (s *Service) ErrWeightMustBePositive() error { return ErrWeightMustBePositive }
func (s *Service) ErrCreativeStatusInvalid() error { return ErrCreativeStatusInvalid }

func (s *Service) MapNotFound(err, target error) error {
	return mapNotFound(err, target)
}

func (s *Service) OnConfigureBrandFcap(ctx context.Context, q db.Querier, brandID uuid.UUID, prev db.AdvertiserBrand, limit, window int32) error {
	payloadBytes, err := coldpath.MarshalOutbox(brandFcapOutboxPayload{
		BrandID:    brandID.String(),
		FreqLimit:  limit,
		FreqWindow: window,
	})
	if err != nil {
		return fmt.Errorf("marshal configure brand fcap outbox payload: %w", err)
	}
	if _, err := q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "CONFIGURE_BRAND_FCAP",
		Payload:   payloadBytes,
	}); err != nil {
		return fmt.Errorf("failed to create outbox event: %w", err)
	}
	s.AuditLog(ctx, q, uuid.Nil, "CONFIGURE_BRAND_FCAP", "brand", &brandID, auditBrandFcapChange{
		OldFreqLimit:  prev.FreqLimit,
		OldFreqWindow: prev.FreqWindow,
		NewFreqLimit:  limit,
		NewFreqWindow: window,
	}, nil)
	return nil
}

func (s *Service) OnBrandCreativesChanged(ctx context.Context, q db.Querier, brandID uuid.UUID) error {
	return s.emitBrandCreativesOutbox(ctx, q, brandID)
}

func (s *Service) CreateBrand(ctx context.Context, customerID uuid.UUID, name string) (uuid.UUID, error) {
	return s.BrandStore().CreateBrand(ctx, customerID, name)
}

func (s *Service) GetBrandDTO(ctx context.Context, id uuid.UUID) (BrandDTO, error) {
	return s.BrandStore().GetBrandDTO(ctx, id)
}

func (s *Service) ListBrandsByCustomer(ctx context.Context, customerID uuid.UUID) ([]BrandDTO, error) {
	return s.BrandStore().ListBrandsByCustomer(ctx, customerID)
}

func (s *Service) ConfigureBrandFcap(ctx context.Context, brandID uuid.UUID, limit, window int32) error {
	return s.BrandStore().ConfigureBrandFcap(ctx, brandID, limit, window)
}

func (s *Service) UpsertBrandCreative(ctx context.Context, brandID uuid.UUID, name, landingURL string, weight int32, status string) (uuid.UUID, error) {
	return s.BrandStore().UpsertBrandCreative(ctx, brandID, name, landingURL, weight, status)
}

func (s *Service) ListBrandCreatives(ctx context.Context, brandID uuid.UUID) ([]BrandCreativeDTO, error) {
	return s.BrandStore().ListBrandCreatives(ctx, brandID)
}

func (s *Service) UpdateBrandCreative(ctx context.Context, creativeID uuid.UUID, name, landingURL string, weight int32, status string) error {
	return s.BrandStore().UpdateBrandCreative(ctx, creativeID, name, landingURL, weight, status)
}

func (s *Service) DeleteBrandCreative(ctx context.Context, creativeID uuid.UUID) error {
	return s.BrandStore().DeleteBrandCreative(ctx, creativeID)
}

func (s *Service) emitBrandCreativesOutbox(ctx context.Context, q db.Querier, brandID uuid.UUID) error {
	payload, err := coldpath.MarshalOutbox(brandIDPayload{BrandID: brandID.String()})
	if err != nil {
		return fmt.Errorf("marshal sync brand creatives outbox payload: %w", err)
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "SYNC_BRAND_CREATIVES", Payload: payload})
	return err
}
