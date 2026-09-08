package campaign

import (
	"context"
	"fmt"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/integrationschema"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func ConversionMappingsFromStatusSchema(s *integrationschema.StatusMappingSchema) ([]ConversionMappingDTO, error) {
	if s == nil || len(s.StatusMap) == 0 {
		return nil, fmt.Errorf("status_map is required")
	}
	out := make([]ConversionMappingDTO, 0, len(s.StatusMap))
	for inbound := range s.StatusMap {
		goal, ok := integrationschema.MapAffiliateStatus(s, inbound)
		if !ok {
			continue
		}
		out = append(out, ConversionMappingDTO{
			InboundStatus: inbound,
			GoalName:      goal,
			PayoutMicro:   0,
		})
	}
	return NormalizeConversionMappings(out)
}

func ReplaceCampaignConversionMappingsTx(
	ctx context.Context,
	q *db.Queries,
	campaignID uuid.UUID,
	mappings []ConversionMappingDTO,
) (int, error) {
	if q == nil {
		return 0, errServiceUnavailable()
	}
	normalized, err := NormalizeConversionMappings(mappings)
	if err != nil {
		return 0, err
	}
	if err := q.DeleteConversionMappingsByCampaign(ctx, domain.ToUUID(campaignID)); err != nil {
		return 0, err
	}
	for i := range normalized {
		row := &normalized[i]
		if err := q.InsertConversionMapping(ctx, db.InsertConversionMappingParams{
			CampaignID:    domain.ToUUID(campaignID),
			InboundStatus: row.InboundStatus,
			GoalName:      row.GoalName,
			PayoutMicro:   row.PayoutMicro,
		}); err != nil {
			return 0, fmt.Errorf("insert conversion mapping: %w", err)
		}
	}
	return len(normalized), nil
}

func ApplyStatusIntegrationSchemaTx(
	ctx context.Context,
	tx pgx.Tx,
	campaignID, schemaID uuid.UUID,
	schemaBody []byte,
) (int, error) {
	if tx == nil {
		return 0, errServiceUnavailable()
	}
	parsedKind, parsed, err := integrationschema.ParseDocument(schemaBody)
	if err != nil || parsedKind != integrationschema.KindStatusMapping {
		return 0, fmt.Errorf("invalid status mapping schema")
	}
	statusSchema := parsed.(*integrationschema.StatusMappingSchema)
	mappings, err := ConversionMappingsFromStatusSchema(statusSchema)
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE campaigns SET status_integration_schema_id = $2, updated_at = NOW() WHERE id = $1`,
		campaignID, schemaID); err != nil {
		return 0, err
	}
	return ReplaceCampaignConversionMappingsTx(ctx, db.New(tx), campaignID, mappings)
}
