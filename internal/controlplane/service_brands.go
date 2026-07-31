package controlplane

import (
	"context"
	"fmt"
	"time"

	"espx/internal/domain"
	"espx/internal/domain/db"
	"espx/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type BrandDTO struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	FreqLimit  int32  `json:"freq_limit"`
	FreqWindow int32  `json:"freq_window"`
}

func toBrandDTO(b db.AdvertiserBrand) BrandDTO {
	return BrandDTO{
		ID:         uuid.UUID(b.ID.Bytes).String(),
		CustomerID: uuid.UUID(b.CustomerID.Bytes).String(),
		Name:       b.Name,
		CreatedAt:  b.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:  b.UpdatedAt.Time.Format(time.RFC3339),
		FreqLimit:  b.FreqLimit,
		FreqWindow: b.FreqWindow,
	}
}

func (s *Service) CreateBrand(ctx context.Context, customerID uuid.UUID, name string) (uuid.UUID, error) {
	brandID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}

	q := db.New(s.GetPool())
	_, err = q.GetCustomerByID(ctx, domain.ToUUID(customerID))
	if err != nil {
		return uuid.Nil, mapNotFound(err, ErrCustomerNotFound)
	}

	_, err = q.CreateBrand(ctx, db.CreateBrandParams{
		ID:         domain.ToUUID(brandID),
		CustomerID: domain.ToUUID(customerID),
		Name:       name,
	})
	if err != nil {
		return uuid.Nil, err
	}

	return brandID, nil
}

func (s *Service) GetBrandDTO(ctx context.Context, id uuid.UUID) (BrandDTO, error) {
	q := db.New(s.GetPool())
	b, err := q.GetBrand(ctx, domain.ToUUID(id))
	if err != nil {
		return BrandDTO{}, mapNotFound(err, ErrBrandNotFound)
	}
	return toBrandDTO(b), nil
}

func (s *Service) ListBrandsByCustomer(ctx context.Context, customerID uuid.UUID) ([]BrandDTO, error) {
	q := db.New(s.GetPool())
	rows, err := q.ListBrandsByCustomer(ctx, domain.ToUUID(customerID))
	if err != nil {
		return nil, err
	}

	return coldpath.MapSlice(rows, toBrandDTO), nil
}

func (s *Service) ConfigureBrandFcap(ctx context.Context, brandID uuid.UUID, limit, window int32) error {
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)

		brand, err := q.GetBrandForUpdate(ctx, domain.ToUUID(brandID))
		if err != nil {
			return mapNotFound(err, ErrBrandNotFound)
		}

		err = q.ConfigureBrandFcap(ctx, db.ConfigureBrandFcapParams{
			ID:         domain.ToUUID(brandID),
			FreqLimit:  limit,
			FreqWindow: window,
		})
		if err != nil {
			return fmt.Errorf("failed to update brand fcap limits: %w", err)
		}

		payloadBytes, err := coldpath.MarshalJSON(map[string]any{
			"brand_id":    brandID.String(),
			"freq_limit":  limit,
			"freq_window": window,
		})
		if err != nil {
			return fmt.Errorf("marshal configure brand fcap outbox payload: %w", err)
		}

		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "CONFIGURE_BRAND_FCAP",
			Payload:   payloadBytes,
		})
		if err != nil {
			return fmt.Errorf("failed to create outbox event: %w", err)
		}

		s.AuditLog(ctx, q, uuid.Nil, "CONFIGURE_BRAND_FCAP", "brand", &brandID, map[string]any{
			"old_freq_limit":  brand.FreqLimit,
			"old_freq_window": brand.FreqWindow,
			"new_freq_limit":  limit,
			"new_freq_window": window,
		}, nil)

		return nil
	})
}
