package flow

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UpdateOfferRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (st *Store) GetOffer(ctx context.Context, offerID uuid.UUID) (OfferDTO, error) {
	if st.poolOrNil() == nil {
		return OfferDTO{}, fmt.Errorf("service unavailable")
	}
	if offerID == uuid.Nil {
		return OfferDTO{}, fmt.Errorf("offer id is required")
	}
	var dto OfferDTO
	err := st.poolOrNil().QueryRow(ctx, `
		SELECT id, name, url, created_at FROM offers WHERE id = $1`, offerID).Scan(
		&dto.ID, &dto.Name, &dto.URL, &dto.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OfferDTO{}, fmt.Errorf("offer not found")
		}
		return OfferDTO{}, err
	}
	return dto, nil
}

func (st *Store) UpdateOffer(ctx context.Context, offerID uuid.UUID, req UpdateOfferRequest) (OfferDTO, error) {
	if st.poolOrNil() == nil {
		return OfferDTO{}, fmt.Errorf("service unavailable")
	}
	if offerID == uuid.Nil {
		return OfferDTO{}, fmt.Errorf("offer id is required")
	}
	name := strings.TrimSpace(req.Name)
	url := strings.TrimSpace(req.URL)
	if name == "" || url == "" {
		return OfferDTO{}, fmt.Errorf("name and url are required")
	}
	tag, err := st.poolOrNil().Exec(ctx, `
		UPDATE offers SET name = $2, url = $3 WHERE id = $1`, offerID, name, url)
	if err != nil {
		return OfferDTO{}, err
	}
	if tag.RowsAffected() == 0 {
		return OfferDTO{}, fmt.Errorf("offer not found")
	}
	return st.GetOffer(ctx, offerID)
}

func (st *Store) DeleteOffer(ctx context.Context, offerID uuid.UUID) error {
	if st.poolOrNil() == nil {
		return fmt.Errorf("service unavailable")
	}
	if offerID == uuid.Nil {
		return fmt.Errorf("offer id is required")
	}
	referenced, err := st.offerReferencedByFlow(ctx, offerID)
	if err != nil {
		return err
	}
	if referenced {
		return fmt.Errorf("offer is referenced by a flow")
	}
	tag, err := st.poolOrNil().Exec(ctx, `DELETE FROM offers WHERE id = $1`, offerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("offer not found")
	}
	return nil
}

func (st *Store) offerReferencedByFlow(ctx context.Context, offerID uuid.UUID) (bool, error) {
	var referenced bool
	err := st.poolOrNil().QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM flows f,
				jsonb_array_elements(f.paths) AS path,
				jsonb_array_elements(path->'offers') AS offer_ref
			WHERE offer_ref->>'offer_id' = $1::text
		)`, offerID).Scan(&referenced)
	return referenced, err
}
