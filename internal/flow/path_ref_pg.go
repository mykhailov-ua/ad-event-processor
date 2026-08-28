package flow

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ValidateLanderIDsPG(ctx context.Context, pool *pgxpool.Pool, ids []uuid.UUID) error {
	if pool == nil || len(ids) == 0 {
		return nil
	}
	rows, err := pool.Query(ctx, `
		SELECT id, COALESCE(url, '') != '' OR hosted_asset_id IS NOT NULL
		FROM landers WHERE id = ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	found := make(map[uuid.UUID]bool, len(ids))
	for rows.Next() {
		var id uuid.UUID
		var routable bool
		if err := rows.Scan(&id, &routable); err != nil {
			return err
		}
		if !routable {
			return fmt.Errorf("lander %s has no URL or hosted asset", id)
		}
		found[id] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		if !found[id] {
			return fmt.Errorf("lander %s not found", id)
		}
	}
	return nil
}

func ValidateOfferIDsPG(ctx context.Context, pool *pgxpool.Pool, ids []uuid.UUID) error {
	if pool == nil || len(ids) == 0 {
		return nil
	}
	rows, err := pool.Query(ctx, `
		SELECT id FROM offers WHERE id = ANY($1) AND COALESCE(url, '') != ''`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	found := make(map[uuid.UUID]struct{}, len(ids))
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		found[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			return fmt.Errorf("offer %s not found", id)
		}
	}
	return nil
}
