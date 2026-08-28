package flow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ad-event-processor/pkg/landerhost"

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

func (st *Store) CreateLander(ctx context.Context, req CreateLanderRequest) (LanderDTO, error) {
	if st.poolOrNil() == nil {
		return LanderDTO{}, fmt.Errorf("service unavailable")
	}
	name := strings.TrimSpace(req.Name)
	url := strings.TrimSpace(req.URL)
	if name == "" {
		return LanderDTO{}, fmt.Errorf("name is required")
	}
	if url == "" {
		url = ""
	}
	var dto LanderDTO
	err := st.poolOrNil().QueryRow(ctx, `
		INSERT INTO landers (name, url) VALUES ($1, NULLIF($2, ''))
		RETURNING id, name, COALESCE(url, ''), hosted_asset_id, created_at`, name, url).Scan(
		&dto.ID, &dto.Name, &dto.URL, &dto.HostedAssetID, &dto.CreatedAt)
	if err != nil {
		return LanderDTO{}, err
	}
	if dto.HostedAssetID != nil {
		dto.HostedURL = landerhost.PublicURL(st.host.LanderPublicBase(ctx), dto.ID)
	}
	return dto, nil
}

func (st *Store) ListLanders(ctx context.Context) ([]LanderDTO, error) {
	if st.poolOrNil() == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	rows, err := st.poolOrNil().Query(ctx, `
		SELECT id, name, COALESCE(url, ''), hosted_asset_id, created_at
		FROM landers ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	publicBase := st.host.LanderPublicBase(ctx)
	var out []LanderDTO
	for rows.Next() {
		var dto LanderDTO
		if err := rows.Scan(&dto.ID, &dto.Name, &dto.URL, &dto.HostedAssetID, &dto.CreatedAt); err != nil {
			return nil, err
		}
		if dto.HostedAssetID != nil {
			dto.HostedURL = landerhost.PublicURL(publicBase, dto.ID)
		}
		out = append(out, dto)
	}
	return out, rows.Err()
}

func (st *Store) CreateOffer(ctx context.Context, req CreateOfferRequest) (OfferDTO, error) {
	if st.poolOrNil() == nil {
		return OfferDTO{}, fmt.Errorf("service unavailable")
	}
	name := strings.TrimSpace(req.Name)
	url := strings.TrimSpace(req.URL)
	if name == "" || url == "" {
		return OfferDTO{}, fmt.Errorf("name and url are required")
	}
	var dto OfferDTO
	err := st.poolOrNil().QueryRow(ctx, `
		INSERT INTO offers (name, url) VALUES ($1, $2)
		RETURNING id, name, url, created_at`, name, url).Scan(&dto.ID, &dto.Name, &dto.URL, &dto.CreatedAt)
	if err != nil {
		return OfferDTO{}, err
	}
	return dto, nil
}

func (st *Store) ListOffers(ctx context.Context) ([]OfferDTO, error) {
	if st.poolOrNil() == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	rows, err := st.poolOrNil().Query(ctx, `SELECT id, name, url, created_at FROM offers ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OfferDTO
	for rows.Next() {
		var dto OfferDTO
		if err := rows.Scan(&dto.ID, &dto.Name, &dto.URL, &dto.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, rows.Err()
}

func (st *Store) CreateFlow(ctx context.Context, req CreateFlowRequest) (DTO, error) {
	if st.poolOrNil() == nil {
		return DTO{}, fmt.Errorf("service unavailable")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return DTO{}, fmt.Errorf("name is required")
	}
	if len(req.Paths) == 0 {
		return DTO{}, fmt.Errorf("paths are required")
	}
	if err := st.host.ValidateFlowPaths(ctx, req.Paths); err != nil {
		return DTO{}, err
	}
	raw, err := json.Marshal(req.Paths)
	if err != nil {
		return DTO{}, err
	}
	var dto DTO
	err = st.poolOrNil().QueryRow(ctx, `
		INSERT INTO flows (name, paths) VALUES ($1, $2::jsonb)
		RETURNING id, name, paths, created_at`, name, raw).Scan(&dto.ID, &dto.Name, &dto.Paths, &dto.CreatedAt)
	if err != nil {
		return DTO{}, err
	}
	return dto, nil
}

func (st *Store) ListFlows(ctx context.Context) ([]DTO, error) {
	if st.poolOrNil() == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	rows, err := st.poolOrNil().Query(ctx, `SELECT id, name, paths, created_at FROM flows ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DTO
	for rows.Next() {
		var dto DTO
		if err := rows.Scan(&dto.ID, &dto.Name, &dto.Paths, &dto.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, rows.Err()
}

func (st *Store) GetFlow(ctx context.Context, flowID uuid.UUID) (DTO, error) {
	if st.poolOrNil() == nil {
		return DTO{}, fmt.Errorf("service unavailable")
	}
	if flowID == uuid.Nil {
		return DTO{}, fmt.Errorf("flow id is required")
	}
	var dto DTO
	err := st.poolOrNil().QueryRow(ctx, `
		SELECT id, name, paths, created_at FROM flows WHERE id = $1`,
		flowID).Scan(&dto.ID, &dto.Name, &dto.Paths, &dto.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DTO{}, fmt.Errorf("flow not found")
		}
		return DTO{}, err
	}
	return dto, nil
}

func (st *Store) UpdateFlow(ctx context.Context, flowID uuid.UUID, req UpdateFlowRequest) (DTO, error) {
	if st.poolOrNil() == nil {
		return DTO{}, fmt.Errorf("service unavailable")
	}
	if flowID == uuid.Nil {
		return DTO{}, fmt.Errorf("flow id is required")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return DTO{}, fmt.Errorf("name is required")
	}
	if len(req.Paths) == 0 {
		return DTO{}, fmt.Errorf("paths are required")
	}
	if err := st.host.ValidateFlowPaths(ctx, req.Paths); err != nil {
		return DTO{}, err
	}
	raw, err := json.Marshal(req.Paths)
	if err != nil {
		return DTO{}, err
	}
	var dto DTO
	err = st.poolOrNil().QueryRow(ctx, `
		UPDATE flows SET name = $2, paths = $3::jsonb, updated_at = now()
		WHERE id = $1
		RETURNING id, name, paths, created_at`, flowID, name, raw).Scan(
		&dto.ID, &dto.Name, &dto.Paths, &dto.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DTO{}, fmt.Errorf("flow not found")
		}
		return DTO{}, err
	}
	_ = st.host.PublishFlowReload(ctx)
	return dto, nil
}
