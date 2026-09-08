package stream

import (
	"context"
	"fmt"

	"ad-event-processor/pkg/affiliatestatus"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type affiliateStatusSchema struct {
	body []byte
}

func (s *affiliateStatusSchema) MapAffiliateStatus(external string) (string, bool) {
	if s == nil {
		return "", false
	}
	return affiliatestatus.MapFromSchemaBody(s.body, external)
}

type PgxAffiliateStatusSchemaStore struct {
	pool *pgxpool.Pool
}

func NewPgxAffiliateStatusSchemaStore(pool *pgxpool.Pool) *PgxAffiliateStatusSchemaStore {
	if pool == nil {
		return nil
	}
	return &PgxAffiliateStatusSchemaStore{pool: pool}
}

func (st *PgxAffiliateStatusSchemaStore) ListAffiliateStatusSchemasByCampaignIDs(
	ctx context.Context,
	ids []pgtype.UUID,
) (map[uuid.UUID]*affiliateStatusSchema, error) {
	if st == nil || st.pool == nil || len(ids) == 0 {
		return nil, nil
	}
	rows, err := st.pool.Query(ctx, `
		SELECT c.id, s.body
		FROM campaigns c
		INNER JOIN integration_schemas s ON s.id = c.status_integration_schema_id
		WHERE c.id = ANY($1::uuid[])
		  AND c.status_integration_schema_id IS NOT NULL`, ids)
	if err != nil {
		return nil, fmt.Errorf("list status integration schemas: %w", err)
	}
	defer rows.Close()

	out := make(map[uuid.UUID]*affiliateStatusSchema)
	for rows.Next() {
		var campID uuid.UUID
		var body []byte
		if err := rows.Scan(&campID, &body); err != nil {
			return nil, err
		}
		if _, ok := affiliatestatus.ParseStatusMappingDocument(body); ok {
			out[campID] = &affiliateStatusSchema{body: body}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
