package brand

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
	host Host
}

func NewStore(pool *pgxpool.Pool, host Host) *Store {
	return &Store{pool: pool, host: host}
}

func (st *Store) poolOrNil() *pgxpool.Pool {
	if st == nil {
		return nil
	}
	return st.pool
}

func (st *Store) CreateBrand(ctx context.Context, customerID uuid.UUID, name string) (uuid.UUID, error) {
	if st.poolOrNil() == nil || st.host == nil {
		return uuid.Nil, fmt.Errorf("service unavailable")
	}
	brandID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}
	q := db.New(st.pool)
	if _, err := q.GetCustomerByID(ctx, domain.ToUUID(customerID)); err != nil {
		return uuid.Nil, st.host.MapNotFound(err, st.host.ErrCustomerNotFound())
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

func (st *Store) GetBrandDTO(ctx context.Context, id uuid.UUID) (DTO, error) {
	if st.poolOrNil() == nil || st.host == nil {
		return DTO{}, fmt.Errorf("service unavailable")
	}
	b, err := db.New(st.pool).GetBrand(ctx, domain.ToUUID(id))
	if err != nil {
		return DTO{}, st.host.MapNotFound(err, st.host.ErrBrandNotFound())
	}
	return brandRowToDTO(b), nil
}

func (st *Store) ListBrandsByCustomer(ctx context.Context, customerID uuid.UUID) ([]DTO, error) {
	if st.poolOrNil() == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	rows, err := db.New(st.pool).ListBrandsByCustomer(ctx, domain.ToUUID(customerID))
	if err != nil {
		return nil, err
	}
	out := make([]DTO, len(rows))
	for i, b := range rows {
		out[i] = brandRowToDTO(b)
	}
	return out, nil
}

func (st *Store) ConfigureBrandFcap(ctx context.Context, brandID uuid.UUID, limit, window int32) error {
	if st.poolOrNil() == nil || st.host == nil {
		return fmt.Errorf("service unavailable")
	}
	return pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		brand, err := q.GetBrandForUpdate(ctx, domain.ToUUID(brandID))
		if err != nil {
			return st.host.MapNotFound(err, st.host.ErrBrandNotFound())
		}
		if err := q.ConfigureBrandFcap(ctx, db.ConfigureBrandFcapParams{
			ID:         domain.ToUUID(brandID),
			FreqLimit:  limit,
			FreqWindow: window,
		}); err != nil {
			return fmt.Errorf("failed to update brand fcap limits: %w", err)
		}
		return st.host.OnConfigureBrandFcap(ctx, q, brandID, brand, limit, window)
	})
}

func (st *Store) UpsertBrandCreative(ctx context.Context, brandID uuid.UUID, name, landingURL string, weight int32, status string) (uuid.UUID, error) {
	if st.poolOrNil() == nil || st.host == nil {
		return uuid.Nil, fmt.Errorf("service unavailable")
	}
	if weight <= 0 {
		return uuid.Nil, st.host.ErrWeightMustBePositive()
	}
	if status == "" {
		status = "ACTIVE"
	}
	if status != "ACTIVE" && status != "PAUSED" {
		return uuid.Nil, st.host.ErrCreativeStatusInvalid()
	}
	creativeID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}
	err = pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		if _, err := q.GetBrand(ctx, domain.ToUUID(brandID)); err != nil {
			return st.host.MapNotFound(err, st.host.ErrBrandNotFound())
		}
		if _, err := q.CreateBrandCreative(ctx, db.CreateBrandCreativeParams{
			ID:         domain.ToUUID(creativeID),
			BrandID:    domain.ToUUID(brandID),
			Name:       name,
			LandingUrl: landingURL,
			Weight:     weight,
			Status:     status,
		}); err != nil {
			return err
		}
		return st.host.OnBrandCreativesChanged(ctx, q, brandID)
	})
	return creativeID, err
}

func (st *Store) ListBrandCreatives(ctx context.Context, brandID uuid.UUID) ([]CreativeDTO, error) {
	if st.poolOrNil() == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	rows, err := db.New(st.pool).ListBrandCreatives(ctx, domain.ToUUID(brandID))
	if err != nil {
		return nil, err
	}
	out := make([]CreativeDTO, len(rows))
	for i, c := range rows {
		out[i] = creativeRowToDTO(c)
	}
	return out, nil
}

func (st *Store) UpdateBrandCreative(ctx context.Context, creativeID uuid.UUID, name, landingURL string, weight int32, status string) error {
	if st.poolOrNil() == nil || st.host == nil {
		return fmt.Errorf("service unavailable")
	}
	return pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		existing, err := q.GetBrandCreative(ctx, domain.ToUUID(creativeID))
		if err != nil {
			return st.host.MapNotFound(err, st.host.ErrCreativeNotFound())
		}
		if _, err := q.UpdateBrandCreative(ctx, db.UpdateBrandCreativeParams{
			ID:         domain.ToUUID(creativeID),
			Name:       name,
			LandingUrl: landingURL,
			Weight:     weight,
			Status:     status,
		}); err != nil {
			return err
		}
		return st.host.OnBrandCreativesChanged(ctx, q, uuid.UUID(existing.BrandID.Bytes))
	})
}

func (st *Store) DeleteBrandCreative(ctx context.Context, creativeID uuid.UUID) error {
	if st.poolOrNil() == nil || st.host == nil {
		return fmt.Errorf("service unavailable")
	}
	return pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		existing, err := q.GetBrandCreative(ctx, domain.ToUUID(creativeID))
		if err != nil {
			return st.host.MapNotFound(err, st.host.ErrCreativeNotFound())
		}
		if err := q.DeleteBrandCreative(ctx, domain.ToUUID(creativeID)); err != nil {
			return err
		}
		return st.host.OnBrandCreativesChanged(ctx, q, uuid.UUID(existing.BrandID.Bytes))
	})
}

func brandRowToDTO(b db.AdvertiserBrand) DTO {
	return DTO{
		ID:         uuid.UUID(b.ID.Bytes).String(),
		CustomerID: uuid.UUID(b.CustomerID.Bytes).String(),
		Name:       b.Name,
		CreatedAt:  b.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:  b.UpdatedAt.Time.Format(time.RFC3339),
		FreqLimit:  b.FreqLimit,
		FreqWindow: b.FreqWindow,
	}
}

func creativeRowToDTO(c db.BrandCreative) CreativeDTO {
	return CreativeDTO{
		ID:         uuid.UUID(c.ID.Bytes).String(),
		BrandID:    uuid.UUID(c.BrandID.Bytes).String(),
		Name:       c.Name,
		LandingURL: c.LandingUrl,
		Weight:     c.Weight,
		Status:     c.Status,
		CreatedAt:  c.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:  c.UpdatedAt.Time.Format(time.RFC3339),
	}
}
