package brand

import (
	"context"
	"fmt"
	"strings"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UpdateRequest struct {
	Name string `json:"name"`
}

func (st *Store) UpdateBrand(ctx context.Context, brandID uuid.UUID, req UpdateRequest) (DTO, error) {
	if st.poolOrNil() == nil || st.host == nil {
		return DTO{}, fmt.Errorf("service unavailable")
	}
	if brandID == uuid.Nil {
		return DTO{}, fmt.Errorf("brand id is required")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return DTO{}, fmt.Errorf("name is required")
	}
	tag, err := st.poolOrNil().Exec(ctx, `
		UPDATE advertiser_brands
		SET name = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`, brandID, name)
	if err != nil {
		return DTO{}, err
	}
	if tag.RowsAffected() == 0 {
		return DTO{}, st.host.ErrBrandNotFound()
	}
	return st.GetBrandDTO(ctx, brandID)
}

func (st *Store) DeleteBrand(ctx context.Context, brandID uuid.UUID) error {
	if st.poolOrNil() == nil || st.host == nil {
		return fmt.Errorf("service unavailable")
	}
	if brandID == uuid.Nil {
		return fmt.Errorf("brand id is required")
	}
	return pgx.BeginFunc(ctx, st.poolOrNil(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if _, err := q.GetBrand(ctx, domain.ToUUID(brandID)); err != nil {
			return st.host.MapNotFound(err, st.host.ErrBrandNotFound())
		}
		referenced, err := brandReferencedByCampaign(ctx, tx, brandID)
		if err != nil {
			return err
		}
		if referenced {
			return fmt.Errorf("brand is referenced by a campaign")
		}
		if _, err := tx.Exec(ctx, `DELETE FROM brand_creatives WHERE brand_id = $1`, brandID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `DELETE FROM advertiser_brands WHERE id = $1`, brandID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return st.host.ErrBrandNotFound()
		}
		return nil
	})
}

func brandReferencedByCampaign(ctx context.Context, tx pgx.Tx, brandID uuid.UUID) (bool, error) {
	var referenced bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM campaigns
			WHERE brand_id = $1 AND deleted_at IS NULL
		)`, brandID).Scan(&referenced)
	return referenced, err
}
