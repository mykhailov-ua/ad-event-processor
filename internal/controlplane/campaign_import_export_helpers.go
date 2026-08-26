package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func parseExportSchedule(startRaw, endRaw string) (*time.Time, *time.Time, error) {
	var startAt, endAt *time.Time
	if strings.TrimSpace(startRaw) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(startRaw))
		if err != nil {
			return nil, nil, fmt.Errorf("invalid start_at")
		}
		startAt = &parsed
	}
	if strings.TrimSpace(endRaw) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(endRaw))
		if err != nil {
			return nil, nil, fmt.Errorf("invalid end_at")
		}
		endAt = &parsed
	}
	return startAt, endAt, nil
}

func defaultTimezone(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "UTC"
	}
	return strings.TrimSpace(raw)
}

func daypartOrEmpty(hours []int16) []int16 {
	if hours == nil {
		return []int16{}
	}
	return hours
}

func defaultTargetEvent(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "conversion"
	}
	return strings.TrimSpace(raw)
}

func (s *Service) integrationSchemaName(ctx context.Context, schemaID uuid.UUID) (string, error) {
	if s == nil || s.pool == nil || schemaID == uuid.Nil {
		return "", nil
	}
	var name string
	err := s.pool.QueryRow(ctx, `SELECT name FROM integration_schemas WHERE id = $1`, schemaID).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}

func (s *Service) resolveIntegrationSchemaID(ctx context.Context, tx pgx.Tx, name string) (pgtype.UUID, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return pgtype.UUID{}, nil
	}
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
SELECT id FROM integration_schemas WHERE name = $1 ORDER BY version DESC LIMIT 1`, name).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgtype.UUID{}, errValidation(fmt.Sprintf("integration schema %q not found", name))
		}
		return pgtype.UUID{}, err
	}
	return domain.ToUUID(id), nil
}

func (s *Service) resolveImportFlowRefs(
	ctx context.Context,
	tx pgx.Tx,
	bundle CampaignExportBundle,
) (map[string]uuid.UUID, map[string]uuid.UUID, []CampaignExportFlowPath, error) {
	if bundle.Flow == nil {
		return nil, nil, nil, nil
	}
	landerByRef := make(map[string]CampaignExportLander, len(bundle.Landers))
	for i := range bundle.Landers {
		row := bundle.Landers[i]
		if row.Ref == "" {
			return nil, nil, nil, errValidation("lander ref is required")
		}
		landerByRef[row.Ref] = row
	}
	offerByRef := make(map[string]CampaignExportOffer, len(bundle.Offers))
	for i := range bundle.Offers {
		row := bundle.Offers[i]
		if row.Ref == "" {
			return nil, nil, nil, errValidation("offer ref is required")
		}
		offerByRef[row.Ref] = row
	}

	landerIDs := make(map[string]uuid.UUID, len(landerByRef))
	for ref, row := range landerByRef {
		id, err := upsertLanderByNameURL(ctx, tx, row.Name, row.URL)
		if err != nil {
			return nil, nil, nil, err
		}
		landerIDs[ref] = id
	}
	offerIDs := make(map[string]uuid.UUID, len(offerByRef))
	for ref, row := range offerByRef {
		id, err := upsertOfferByNameURL(ctx, tx, row.Name, row.URL)
		if err != nil {
			return nil, nil, nil, err
		}
		offerIDs[ref] = id
	}
	return landerIDs, offerIDs, bundle.Flow.Paths, nil
}

func upsertLanderByNameURL(ctx context.Context, tx pgx.Tx, name, url string) (uuid.UUID, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return uuid.Nil, errValidation("lander name is required")
	}
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
SELECT id FROM landers WHERE name = $1 AND COALESCE(url, '') = $2 LIMIT 1`, name, strings.TrimSpace(url)).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	err = tx.QueryRow(ctx, `
INSERT INTO landers (name, url) VALUES ($1, NULLIF($2, '')) RETURNING id`, name, strings.TrimSpace(url)).Scan(&id)
	return id, err
}

func upsertOfferByNameURL(ctx context.Context, tx pgx.Tx, name, url string) (uuid.UUID, error) {
	name = strings.TrimSpace(name)
	url = strings.TrimSpace(url)
	if name == "" || url == "" {
		return uuid.Nil, errValidation("offer name and url are required")
	}
	var id uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM offers WHERE name = $1 AND url = $2 LIMIT 1`, name, url).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	err = tx.QueryRow(ctx, `INSERT INTO offers (name, url) VALUES ($1, $2) RETURNING id`, name, url).Scan(&id)
	return id, err
}

func (s *Service) validateFlowPathsWithIDs(
	ctx context.Context,
	tx pgx.Tx,
	paths []CampaignExportFlowPath,
	landerIDs map[string]uuid.UUID,
	offerIDs map[string]uuid.UUID,
) error {
	flowPaths := remapFlowPathsForInsert(paths, landerIDs, offerIDs)
	if err := validateFlowPathShape(flowPaths); err != nil {
		return err
	}
	ids := make([]uuid.UUID, 0, len(landerIDs))
	for _, id := range landerIDs {
		ids = append(ids, id)
	}
	if err := validateFlowLanderIDsTx(ctx, tx, ids); err != nil {
		return err
	}
	ids = ids[:0]
	for _, id := range offerIDs {
		ids = append(ids, id)
	}
	return validateFlowOfferIDsTx(ctx, tx, ids)
}

func remapFlowPathsForInsert(
	paths []CampaignExportFlowPath,
	landerIDs map[string]uuid.UUID,
	offerIDs map[string]uuid.UUID,
) []FlowPathDTO {
	out := make([]FlowPathDTO, 0, len(paths))
	for _, path := range paths {
		fp := FlowPathDTO{Weight: path.Weight, Filters: path.Filters}
		for _, lander := range path.Landers {
			fp.Landers = append(fp.Landers, FlowPathLanderRef{
				LanderID: landerIDs[lander.Ref],
				Weight:   lander.Weight,
			})
		}
		for _, offer := range path.Offers {
			fp.Offers = append(fp.Offers, FlowPathOfferRef{
				OfferID:  offerIDs[offer.Ref],
				Weight:   offer.Weight,
				CapDaily: offer.CapDaily,
				CapTotal: offer.CapTotal,
			})
		}
		out = append(out, fp)
	}
	return out
}

func validateFlowLanderIDsTx(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := tx.Query(ctx, `
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

func validateFlowOfferIDsTx(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := tx.Query(ctx, `SELECT id FROM offers WHERE id = ANY($1)`, ids)
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

func applyCampaignIngressCostPatchTx(
	ctx context.Context,
	q *db.Queries,
	campaignID uuid.UUID,
	cfg IngressCostConfigDTO,
) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("invalid ingress_cost_config")
	}
	parsed := domain.ParseIngressCostConfigJSON(raw)
	if cfg.Param != "" && !parsed.Enabled() {
		return fmt.Errorf("invalid ingress_cost_config.param")
	}
	if parsed.MaxMicro < 0 {
		return fmt.Errorf("invalid ingress_cost_config.max_micro")
	}
	_, err = q.UpdateCampaignIngressCostConfig(ctx, db.UpdateCampaignIngressCostConfigParams{
		ID:                domain.ToUUID(campaignID),
		IngressCostConfig: raw,
	})
	return err
}

func ingressCostConfigDTOFromRaw(raw []byte) *IngressCostConfigDTO {
	if len(raw) == 0 {
		return nil
	}
	parsed := domain.ParseIngressCostConfigJSON(raw)
	if !parsed.Enabled() {
		return nil
	}
	scale := "decimal"
	if parsed.ScaleMicro {
		scale = "micro"
	}
	policy := "ignore"
	if parsed.Policy == domain.IngressCostPolicyReject {
		policy = "reject"
	}
	param := ""
	switch parsed.Param {
	case domain.IngressCostParamCost:
		param = "cost"
	case domain.IngressCostParamCPC:
		param = "cpc"
	case domain.IngressCostParamBid:
		param = "bid"
	}
	return &IngressCostConfigDTO{
		Param:    param,
		Scale:    scale,
		MaxMicro: parsed.MaxMicro,
		Policy:   policy,
	}
}
