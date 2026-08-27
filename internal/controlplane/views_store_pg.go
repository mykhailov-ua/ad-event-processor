package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *ViewsStore) pgEnabled() bool {
	return s != nil && s.pool != nil
}

func (s *ViewsStore) createViewPG(ctx context.Context, req CreateViewRequest, ownerID string) (SavedViewDTO, error) {
	spec := req.Spec
	if len(spec) == 0 {
		spec = json.RawMessage(`{}`)
	}
	var id uuid.UUID
	var createdAt, updatedAt time.Time
	err := s.pool.QueryRow(ctx, `
INSERT INTO report_saved_views (owner_id, customer_id, name, report_key, spec, is_shared)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at, updated_at`,
		ownerID,
		uuid.MustParse(req.CustomerID),
		req.Name,
		req.ReportKey,
		spec,
		req.IsShared,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return SavedViewDTO{}, fmt.Errorf("insert saved view: %w", err)
	}
	return SavedViewDTO{
		ID:         id.String(),
		OwnerID:    ownerID,
		CustomerID: req.CustomerID,
		Name:       req.Name,
		ReportKey:  req.ReportKey,
		Spec:       spec,
		IsShared:   req.IsShared,
		CreatedAt:  createdAt.UTC().Format(time.RFC3339),
		UpdatedAt:  updatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (s *ViewsStore) getViewPG(ctx context.Context, id string) (SavedViewDTO, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return SavedViewDTO{}, ErrViewNotFound
	}
	var view SavedViewDTO
	var customerID uuid.UUID
	var specJSON []byte
	var createdAt, updatedAt time.Time
	err = s.pool.QueryRow(ctx, `
SELECT id, owner_id, customer_id, name, report_key, spec, is_shared, created_at, updated_at
FROM report_saved_views
WHERE id = $1`, parsed).Scan(
		&parsed, &view.OwnerID, &customerID, &view.Name, &view.ReportKey, &specJSON, &view.IsShared, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SavedViewDTO{}, ErrViewNotFound
		}
		return SavedViewDTO{}, err
	}
	view.ID = parsed.String()
	view.CustomerID = customerID.String()
	view.Spec = specJSON
	view.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	view.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return view, nil
}

func (s *ViewsStore) listViewsPG(ctx context.Context, customerID string) ([]SavedViewDTO, error) {
	cid, err := uuid.Parse(customerID)
	if err != nil {
		return nil, fmt.Errorf("invalid customer_id")
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, owner_id, customer_id, name, report_key, spec, is_shared, created_at, updated_at
FROM report_saved_views
WHERE customer_id = $1
ORDER BY updated_at DESC`, cid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]SavedViewDTO, 0, 8)
	for rows.Next() {
		var view SavedViewDTO
		var id, rowCustomerID uuid.UUID
		var specJSON []byte
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &view.OwnerID, &rowCustomerID, &view.Name, &view.ReportKey, &specJSON, &view.IsShared, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		view.ID = id.String()
		view.CustomerID = rowCustomerID.String()
		view.Spec = specJSON
		view.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		view.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		out = append(out, view)
	}
	return out, rows.Err()
}

func (s *ViewsStore) updateViewPG(ctx context.Context, id string, req UpdateViewRequest) (SavedViewDTO, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return SavedViewDTO{}, ErrViewNotFound
	}
	spec := req.Spec
	if len(spec) == 0 {
		spec = json.RawMessage(`{}`)
	}
	tag, err := s.pool.Exec(ctx, `
UPDATE report_saved_views
SET name = $2, report_key = $3, spec = $4, is_shared = $5, updated_at = NOW()
WHERE id = $1`,
		parsed, req.Name, req.ReportKey, spec, req.IsShared,
	)
	if err != nil {
		return SavedViewDTO{}, err
	}
	if tag.RowsAffected() == 0 {
		return SavedViewDTO{}, ErrViewNotFound
	}
	return s.getViewPG(ctx, id)
}

func (s *ViewsStore) deleteViewPG(ctx context.Context, id string) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return ErrViewNotFound
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM report_saved_views WHERE id = $1`, parsed)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrViewNotFound
	}
	return nil
}
