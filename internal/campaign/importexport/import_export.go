package importexport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"ad-event-processor/internal/campaign"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/migrationsource"
	"ad-event-processor/internal/reportjob"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ExportCampaign(ctx context.Context, host campaign.ImportExportHost, campaignID uuid.UUID) (campaign.CampaignExportBundle, error) {
	if host == nil || host.Pool() == nil {
		return campaign.CampaignExportBundle{}, fmt.Errorf("service unavailable")
	}
	row, err := db.New(host.Pool()).GetCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return campaign.CampaignExportBundle{}, campaign.MapCampaignNotFound(err, campaign.ErrCampaignNotFound)
	}
	if err := host.AssertMediaBuyerCampaignAccess(ctx, row); err != nil {
		return campaign.CampaignExportBundle{}, err
	}
	if row.DeletedAt.Valid {
		return campaign.CampaignExportBundle{}, campaign.ErrCampaignNotFound
	}

	bundle := campaign.CampaignExportBundle{
		ExportVersion: CampaignExportVersion,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		Campaign:      campaignRowToExport(row),
	}

	if row.FlowID.Valid {
		flowID := uuid.UUID(row.FlowID.Bytes)
		flow, flowErr := host.GetFlow(ctx, flowID)
		if flowErr != nil {
			return campaign.CampaignExportBundle{}, flowErr
		}
		exportFlow, landerRefs, offerRefs, convErr := exportFlowBundle(flow)
		if convErr != nil {
			return campaign.CampaignExportBundle{}, convErr
		}
		bundle.Flow = exportFlow
		if err := enrichExportFlowAssets(ctx, host.Pool(), &bundle, landerRefs, offerRefs); err != nil {
			return campaign.CampaignExportBundle{}, err
		}
	}

	q := db.New(host.Pool())
	if pb, err := q.GetPostbackConfig(ctx, domain.ToUUID(campaignID)); err == nil {
		bundle.PostbackConfig = &campaign.CampaignExportPostback{
			Provider:      pb.Provider,
			URLTemplate:   pb.UrlTemplate,
			TargetEvent:   pb.TargetEvent,
			TestEventCode: pb.TestEventCode,
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return campaign.CampaignExportBundle{}, err
	}

	mappings, err := q.ListConversionMappingsByCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return campaign.CampaignExportBundle{}, err
	}
	for i := range mappings {
		bundle.ConversionMappings = append(bundle.ConversionMappings, campaign.ConversionMappingToDTO(&mappings[i]))
	}

	if row.IntegrationSchemaID.Valid {
		bundle.IntegrationSchemaName, _ = integrationSchemaName(ctx, host.Pool(), uuid.UUID(row.IntegrationSchemaID.Bytes))
	}
	if row.StatusIntegrationSchemaID.Valid {
		bundle.StatusIntegrationSchemaName, _ = integrationSchemaName(ctx, host.Pool(), uuid.UUID(row.StatusIntegrationSchemaID.Bytes))
	}

	raw, err := json.Marshal(bundle)
	if err != nil {
		return campaign.CampaignExportBundle{}, err
	}
	if len(raw) > MaxCampaignImportBytes {
		return campaign.CampaignExportBundle{}, fmt.Errorf("export bundle exceeds %d bytes", MaxCampaignImportBytes)
	}
	return bundle, nil
}

func ImportCampaign(ctx context.Context, host campaign.ImportExportHost, spec campaign.ImportCampaignSpec) (campaign.ImportCampaignResult, error) {
	if host == nil || host.Pool() == nil {
		return campaign.ImportCampaignResult{}, fmt.Errorf("service unavailable")
	}
	if spec.CustomerID == uuid.Nil {
		return campaign.ImportCampaignResult{}, campaign.ErrValidationf("customer_id is required")
	}
	if strings.TrimSpace(spec.IdempotencyKey) == "" {
		return campaign.ImportCampaignResult{}, campaign.ErrValidationf("idempotency key is required")
	}
	if spec.Bundle.ExportVersion != CampaignExportVersion {
		return campaign.ImportCampaignResult{}, campaign.ErrValidationf("unsupported export_version")
	}
	raw, err := json.Marshal(spec.Bundle)
	if err != nil {
		return campaign.ImportCampaignResult{}, err
	}
	if len(raw) > MaxCampaignImportBytes {
		return campaign.ImportCampaignResult{}, campaign.ErrValidationf(fmt.Sprintf("import bundle exceeds %d bytes", MaxCampaignImportBytes))
	}

	camp := spec.Bundle.Campaign
	name := strings.TrimSpace(camp.Name)
	if spec.NameOverride != "" {
		name = strings.TrimSpace(spec.NameOverride)
	}
	if name == "" {
		return campaign.ImportCampaignResult{}, campaign.ErrValidationf("campaign name is required")
	}
	budget := camp.BudgetLimitMicro
	if spec.BudgetOverride != nil {
		budget = *spec.BudgetOverride
	}
	if budget <= 0 {
		return campaign.ImportCampaignResult{}, campaign.ErrValidationf("budget_limit_micro must be positive")
	}

	startAt, endAt, err := parseExportSchedule(camp.StartAt, camp.EndAt)
	if err != nil {
		return campaign.ImportCampaignResult{}, campaign.ErrValidationf(err.Error())
	}
	if err := campaign.ValidateDaypartHours(camp.DaypartHours); err != nil {
		return campaign.ImportCampaignResult{}, campaign.ErrValidationf(err.Error())
	}
	if err := campaign.ValidateSchedule(startAt, endAt); err != nil {
		return campaign.ImportCampaignResult{}, campaign.ErrValidationf(err.Error())
	}

	var (
		newCampaignID uuid.UUID
		importedName  string
		flowImported  bool
	)
	err = pgx.BeginFunc(ctx, host.Pool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		existing, err := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: spec.IdempotencyKey, Valid: true})
		if err == nil {
			if existing.CampaignID.Valid {
				newCampaignID = uuid.UUID(existing.CampaignID.Bytes)
				row, err := q.GetCampaign(ctx, existing.CampaignID)
				if err != nil {
					return err
				}
				importedName = row.Name
				return nil
			}
			return fmt.Errorf("%w ledger row for key %q", campaign.ErrIncompleteIdempotency, spec.IdempotencyKey)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("idempotency lookup failed: %w", err)
		}

		cust, err := q.GetCustomerForUpdate(ctx, domain.ToUUID(spec.CustomerID))
		if err != nil {
			return campaign.MapCampaignNotFound(err, campaign.ErrCustomerNotFound)
		}
		if cust.Balance+cust.AllowedOverdraft < budget {
			return campaign.ErrInsufficientBalance
		}

		newCampaignID, err = uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate campaign id: %w", err)
		}
		importedName = name

		landerIDs, offerIDs, flowPaths, err := resolveImportFlowRefs(ctx, tx, spec.Bundle)
		if err != nil {
			return err
		}
		if len(flowPaths) > 0 {
			if err := validateFlowPathsWithIDs(ctx, tx, flowPaths, landerIDs, offerIDs); err != nil {
				return err
			}
		}

		var flowID pgtype.UUID
		if spec.Bundle.Flow != nil && len(flowPaths) > 0 {
			flowUUID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generate flow id: %w", err)
			}
			rawPaths, err := json.Marshal(remapFlowPathsForInsert(flowPaths, landerIDs, offerIDs))
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO flows (id, name, paths) VALUES ($1, $2, $3::jsonb)`,
				flowUUID, strings.TrimSpace(spec.Bundle.Flow.Name), rawPaths); err != nil {
				return fmt.Errorf("insert flow: %w", err)
			}
			flowID = domain.ToUUID(flowUUID)
			flowImported = true
		}

		pacing := db.PacingModeTypeASAP
		if camp.PacingMode != "" {
			pacing = db.PacingModeType(camp.PacingMode)
		}
		initialStatus := campaign.ResolveScheduleStatus(time.Now(), startAt, endAt)
		brandFcapKey := "fcap:c:" + newCampaignID.String()
		attestationMode := strings.TrimSpace(camp.AttestationMode)
		if attestationMode == "" {
			attestationMode = "off"
		}

		if _, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      domain.ToUUID(spec.CustomerID),
			Balance: -budget,
		}); err != nil {
			return err
		}

		if _, err = q.CreateCampaign(ctx, db.CreateCampaignParams{
			ID:              domain.ToUUID(newCampaignID),
			Name:            importedName,
			BudgetLimit:     budget,
			Status:          initialStatus,
			CustomerID:      domain.ToUUID(spec.CustomerID),
			PacingMode:      pacing,
			DailyBudget:     camp.DailyBudgetMicro,
			Timezone:        DefaultTimezone(camp.Timezone),
			FreqLimit:       pgtype.Int4{Int32: camp.FreqLimit, Valid: camp.FreqLimit > 0},
			FreqWindow:      pgtype.Int4{Int32: camp.FreqWindow, Valid: camp.FreqWindow > 0},
			TargetCountries: campaign.CountriesOrEmpty(camp.TargetCountries),
			BrandFcapKey:    brandFcapKey,
			StartAt:         campaign.ToTimestamptz(startAt),
			EndAt:           campaign.ToTimestamptz(endAt),
			DaypartHours:    campaign.DaypartOrEmpty(camp.DaypartHours),
		}); err != nil {
			return err
		}

		if _, err = q.UpdateCampaignAdmin(ctx, db.UpdateCampaignAdminParams{
			ID:                         domain.ToUUID(newCampaignID),
			Name:                       importedName,
			DailyBudget:                camp.DailyBudgetMicro,
			Timezone:                   DefaultTimezone(camp.Timezone),
			FreqLimit:                  pgtype.Int4{Int32: camp.FreqLimit, Valid: camp.FreqLimit > 0},
			FreqWindow:                 pgtype.Int4{Int32: camp.FreqWindow, Valid: camp.FreqWindow > 0},
			TargetCountries:            campaign.CountriesOrEmpty(camp.TargetCountries),
			TargetUrl:                  camp.TargetURL,
			ReferrerFilter:             camp.ReferrerFilter,
			SafePageUrl:                camp.SafePageURL,
			SafePageEnabled:            camp.SafePageEnabled,
			AttestationEnabled:         camp.AttestationEnabled,
			AttestationTtlSec:          camp.AttestationTTLSec,
			AttestationMode:            attestationMode,
			DmrEnabled:                 camp.DmrEnabled,
			ClickDelivery:              camp.ClickDelivery,
			ProxyUpstreamUrl:           camp.ProxyUpstreamURL,
			ProxyRewriteAssets:         camp.ProxyRewriteAssets,
			TlsFingerprintBlockEnabled: camp.TLSFingerprintBlockEnabled,
			ConnTypePolicy:             camp.ConnTypePolicy,
			LinkSigningEnabled:         camp.LinkSigningEnabled,
			LinkSigningTtlSec:          camp.LinkSigningTTLSec,
			CidrBlockEnabled:           camp.CIDRBlockEnabled,
			ProxyVpnBlockEnabled:       camp.ProxyVPNBlockEnabled,
			ModeratorIntelEnabled:      camp.ModeratorIntelEnabled,
			ReviewTrafficAction:        camp.ReviewTrafficAction,
		}); err != nil {
			return err
		}

		if _, err = q.UpdateCampaignFraudConfig(ctx, db.UpdateCampaignFraudConfigParams{
			ID:                    domain.ToUUID(newCampaignID),
			FraudThresholdPass:    camp.FraudThresholdPass,
			FraudThresholdSuspect: camp.FraudThresholdSuspect,
			FraudThresholdIvt:     camp.FraudThresholdIVT,
			FraudThresholdBlock:   camp.FraudThresholdBlock,
			SilentRejectEnabled:   camp.SilentRejectEnabled,
			BehaviorFlags:         0,
			CanvasRetestEnabled:   false,
			CgnatIpPolicyEnabled:  false,
			ConversionRejectRules: []byte("{}"),
		}); err != nil {
			return err
		}

		if len(camp.CreativePayload) > 0 {
			if _, err := tx.Exec(ctx, `UPDATE campaigns SET creative_payload = $2::jsonb WHERE id = $1`, newCampaignID, camp.CreativePayload); err != nil {
				return err
			}
		}

		integrationID, err := resolveIntegrationSchemaID(ctx, tx, spec.Bundle.IntegrationSchemaName)
		if err != nil {
			return err
		}
		statusIntegrationID, err := resolveIntegrationSchemaID(ctx, tx, spec.Bundle.StatusIntegrationSchemaName)
		if err != nil {
			return err
		}
		if integrationID.Valid || statusIntegrationID.Valid || flowID.Valid {
			if _, err := tx.Exec(ctx, `
UPDATE campaigns SET integration_schema_id = $2, status_integration_schema_id = $3, flow_id = $4
WHERE id = $1`, newCampaignID, integrationID, statusIntegrationID, flowIDOrNil(uuid.UUID(flowID.Bytes))); err != nil {
				return err
			}
		}

		if camp.IngressCostConfig != nil {
			if err := applyCampaignIngressCostPatchTx(ctx, q, newCampaignID, *camp.IngressCostConfig); err != nil {
				return err
			}
		}

		if err := campaign.ApplyCampaignClickPresetTx(ctx, tx, newCampaignID, camp.TrafficTemplateID, camp.ClickQueryParams); err != nil {
			return fmt.Errorf("apply click preset: %w", err)
		}

		if spec.Bundle.PostbackConfig != nil {
			pb := spec.Bundle.PostbackConfig
			if err := q.UpsertPostbackConfig(ctx, db.UpsertPostbackConfigParams{
				CampaignID:        domain.ToUUID(newCampaignID),
				Provider:          pb.Provider,
				UrlTemplate:       pb.URLTemplate,
				ApiTokenEncrypted: []byte{},
				TargetEvent:       defaultTargetEvent(pb.TargetEvent),
				TestEventCode:     pb.TestEventCode,
			}); err != nil {
				return fmt.Errorf("upsert postback config: %w", err)
			}
		}

		normalized, err := campaign.NormalizeConversionMappings(spec.Bundle.ConversionMappings)
		if err != nil {
			return campaign.ErrValidationf(err.Error())
		}
		for i := range normalized {
			row := &normalized[i]
			if err := q.InsertConversionMapping(ctx, db.InsertConversionMappingParams{
				CampaignID:    domain.ToUUID(newCampaignID),
				InboundStatus: row.InboundStatus,
				GoalName:      row.GoalName,
				PayoutMicro:   row.PayoutMicro,
			}); err != nil {
				return err
			}
		}

		_, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      domain.ToUUID(spec.CustomerID),
			CampaignID:      domain.ToUUID(newCampaignID),
			Amount:          budget,
			Type:            db.LedgerTypeFREEZE,
			IdempotencyHash: pgtype.Text{String: spec.IdempotencyKey, Valid: true},
		})
		if err != nil {
			return err
		}

		if err := q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
			CampaignID: domain.ToUUID(newCampaignID),
			NewStatus:  initialStatus,
			Reason:     pgtype.Text{String: "Imported campaign bundle", Valid: true},
		}); err != nil {
			return err
		}

		if err := host.AuditImportCampaign(ctx, q, newCampaignID, campaign.ImportCampaignAuditChange{Name: importedName}, campaign.ImportCampaignIdempotencyMeta{IdempotencyKey: spec.IdempotencyKey}); err != nil {
			return err
		}

		return host.EmitCampaignLifecycleOutbox(ctx, q, newCampaignID, initialStatus, budget)
	})
	if err != nil {
		return campaign.ImportCampaignResult{}, err
	}

	host.PublishCampaignUpdate(ctx, newCampaignID.String())
	if flowImported {
		host.PublishFlowReload(ctx)
	}

	return campaign.ImportCampaignResult{ID: newCampaignID.String(), Name: importedName}, nil
}

func campaignRowToExport(row db.Campaign) campaign.CampaignExportCampaign {
	out := campaign.CampaignExportCampaign{
		Name:                       row.Name,
		BudgetLimitMicro:           row.BudgetLimit,
		PacingMode:                 string(row.PacingMode),
		DailyBudgetMicro:           row.DailyBudget,
		Timezone:                   row.Timezone,
		TargetCountries:            append([]string(nil), row.TargetCountries...),
		TargetURL:                  row.TargetUrl,
		SafePageURL:                row.SafePageUrl,
		SafePageEnabled:            row.SafePageEnabled,
		AttestationEnabled:         row.AttestationEnabled,
		AttestationMode:            row.AttestationMode,
		AttestationTTLSec:          row.AttestationTtlSec,
		DmrEnabled:                 row.DmrEnabled,
		CIDRBlockEnabled:           row.CidrBlockEnabled,
		ProxyVPNBlockEnabled:       row.ProxyVpnBlockEnabled,
		ModeratorIntelEnabled:      row.ModeratorIntelEnabled,
		ReviewTrafficAction:        row.ReviewTrafficAction,
		TLSFingerprintBlockEnabled: row.TlsFingerprintBlockEnabled,
		ConnTypePolicy:             row.ConnTypePolicy,
		LinkSigningEnabled:         row.LinkSigningEnabled,
		LinkSigningTTLSec:          row.LinkSigningTtlSec,
		ClickDelivery:              row.ClickDelivery,
		ProxyUpstreamURL:           row.ProxyUpstreamUrl,
		ProxyRewriteAssets:         row.ProxyRewriteAssets,
		ReferrerFilter:             row.ReferrerFilter,
		DaypartHours:               append([]int16(nil), row.DaypartHours...),
		FraudThresholdPass:         row.FraudThresholdPass,
		FraudThresholdSuspect:      row.FraudThresholdSuspect,
		FraudThresholdIVT:          row.FraudThresholdIvt,
		FraudThresholdBlock:        row.FraudThresholdBlock,
		SilentRejectEnabled:        row.SilentRejectEnabled,
	}
	if row.FreqLimit.Valid {
		out.FreqLimit = row.FreqLimit.Int32
	}
	if row.FreqWindow.Valid {
		out.FreqWindow = row.FreqWindow.Int32
	}
	if row.StartAt.Valid {
		out.StartAt = row.StartAt.Time.UTC().Format(time.RFC3339)
	}
	if row.EndAt.Valid {
		out.EndAt = row.EndAt.Time.UTC().Format(time.RFC3339)
	}
	if len(row.CreativePayload) > 0 {
		out.CreativePayload = json.RawMessage(append([]byte(nil), row.CreativePayload...))
	}
	out.IngressCostConfig = ingressCostConfigDTOFromRaw(row.IngressCostConfig)
	out.TrafficTemplateID = campaign.FormatOptionalText(row.TrafficTemplateID)
	out.ClickQueryParams = campaign.ClickQueryParamsFromRaw(row.ClickQueryParams)
	return out
}

func exportFlowBundle(flow campaign.FlowDTO) (*campaign.CampaignExportFlow, map[uuid.UUID]campaign.CampaignExportLander, map[uuid.UUID]campaign.CampaignExportOffer, error) {
	var paths []campaign.FlowPathDTO
	if len(flow.Paths) > 0 {
		if err := json.Unmarshal(flow.Paths, &paths); err != nil {
			return nil, nil, nil, fmt.Errorf("parse flow paths: %w", err)
		}
	}
	landerByID := make(map[uuid.UUID]campaign.CampaignExportLander)
	offerByID := make(map[uuid.UUID]campaign.CampaignExportOffer)
	exportPaths := make([]campaign.CampaignExportFlowPath, 0, len(paths))
	for _, path := range paths {
		ep := campaign.CampaignExportFlowPath{Weight: path.Weight, Filters: path.Filters}
		for _, lander := range path.Landers {
			ref := lander.LanderID.String()
			landerByID[lander.LanderID] = campaign.CampaignExportLander{Ref: ref}
			ep.Landers = append(ep.Landers, campaign.CampaignExportFlowLanderRef{Ref: ref, Weight: lander.Weight})
		}
		for _, offer := range path.Offers {
			ref := offer.OfferID.String()
			offerByID[offer.OfferID] = campaign.CampaignExportOffer{Ref: ref}
			ep.Offers = append(ep.Offers, campaign.CampaignExportFlowOfferRef{
				Ref:      ref,
				Weight:   offer.Weight,
				CapDaily: offer.CapDaily,
				CapTotal: offer.CapTotal,
			})
		}
		exportPaths = append(exportPaths, ep)
	}
	return &campaign.CampaignExportFlow{Name: flow.Name, Paths: exportPaths}, landerByID, offerByID, nil
}

func enrichExportFlowAssets(ctx context.Context, pool *pgxpool.Pool, bundle *campaign.CampaignExportBundle, landerRefs map[uuid.UUID]campaign.CampaignExportLander, offerRefs map[uuid.UUID]campaign.CampaignExportOffer) error {
	if bundle == nil || pool == nil {
		return nil
	}
	landerIDs := make([]uuid.UUID, 0, len(landerRefs))
	for id := range landerRefs {
		landerIDs = append(landerIDs, id)
	}
	if len(landerIDs) > 0 {
		rows, err := pool.Query(ctx, `SELECT id, name, COALESCE(url, '') FROM landers WHERE id = ANY($1)`, landerIDs)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id uuid.UUID
			var name, url string
			if err := rows.Scan(&id, &name, &url); err != nil {
				return err
			}
			ref := landerRefs[id]
			ref.Name = name
			ref.URL = url
			landerRefs[id] = ref
		}
		if err := rows.Err(); err != nil {
			return err
		}
	}
	offerIDs := make([]uuid.UUID, 0, len(offerRefs))
	for id := range offerRefs {
		offerIDs = append(offerIDs, id)
	}
	if len(offerIDs) > 0 {
		rows, err := pool.Query(ctx, `SELECT id, name, url FROM offers WHERE id = ANY($1)`, offerIDs)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id uuid.UUID
			var name, url string
			if err := rows.Scan(&id, &name, &url); err != nil {
				return err
			}
			ref := offerRefs[id]
			ref.Name = name
			ref.URL = url
			offerRefs[id] = ref
		}
		if err := rows.Err(); err != nil {
			return err
		}
	}
	bundle.Landers = make([]campaign.CampaignExportLander, 0, len(landerRefs))
	for _, row := range landerRefs {
		bundle.Landers = append(bundle.Landers, row)
	}
	bundle.Offers = make([]campaign.CampaignExportOffer, 0, len(offerRefs))
	for _, row := range offerRefs {
		bundle.Offers = append(bundle.Offers, row)
	}
	return nil
}

const (
	CampaignExportVersion  = 1
	MaxCampaignImportBytes = 64 * 1024
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

func DefaultTimezone(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "UTC"
	}
	return strings.TrimSpace(raw)
}

func defaultTargetEvent(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "conversion"
	}
	return strings.TrimSpace(raw)
}

func integrationSchemaName(ctx context.Context, pool *pgxpool.Pool, schemaID uuid.UUID) (string, error) {
	if pool == nil || schemaID == uuid.Nil {
		return "", nil
	}
	var name string
	err := pool.QueryRow(ctx, `SELECT name FROM integration_schemas WHERE id = $1`, schemaID).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}

func resolveIntegrationSchemaID(ctx context.Context, tx pgx.Tx, name string) (pgtype.UUID, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return pgtype.UUID{}, nil
	}
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
SELECT id FROM integration_schemas WHERE name = $1 ORDER BY version DESC LIMIT 1`, name).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgtype.UUID{}, campaign.ErrValidationf(fmt.Sprintf("integration schema %q not found", name))
		}
		return pgtype.UUID{}, err
	}
	return domain.ToUUID(id), nil
}

func resolveImportFlowRefs(
	ctx context.Context,
	tx pgx.Tx,
	bundle campaign.CampaignExportBundle,
) (map[string]uuid.UUID, map[string]uuid.UUID, []campaign.CampaignExportFlowPath, error) {
	if bundle.Flow == nil {
		return nil, nil, nil, nil
	}
	landerByRef := make(map[string]campaign.CampaignExportLander, len(bundle.Landers))
	for i := range bundle.Landers {
		row := bundle.Landers[i]
		if row.Ref == "" {
			return nil, nil, nil, campaign.ErrValidationf("lander ref is required")
		}
		landerByRef[row.Ref] = row
	}
	offerByRef := make(map[string]campaign.CampaignExportOffer, len(bundle.Offers))
	for i := range bundle.Offers {
		row := bundle.Offers[i]
		if row.Ref == "" {
			return nil, nil, nil, campaign.ErrValidationf("offer ref is required")
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
		return uuid.Nil, campaign.ErrValidationf("lander name is required")
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
		return uuid.Nil, campaign.ErrValidationf("offer name and url are required")
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

func validateFlowPathsWithIDs(
	ctx context.Context,
	tx pgx.Tx,
	paths []campaign.CampaignExportFlowPath,
	landerIDs map[string]uuid.UUID,
	offerIDs map[string]uuid.UUID,
) error {
	flowPaths := remapFlowPathsForInsert(paths, landerIDs, offerIDs)
	if err := campaign.ValidateFlowPathShape(flowPaths); err != nil {
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
	paths []campaign.CampaignExportFlowPath,
	landerIDs map[string]uuid.UUID,
	offerIDs map[string]uuid.UUID,
) []campaign.FlowPathDTO {
	out := make([]campaign.FlowPathDTO, 0, len(paths))
	for _, path := range paths {
		fp := campaign.FlowPathDTO{Weight: path.Weight, Filters: path.Filters}
		for _, lander := range path.Landers {
			fp.Landers = append(fp.Landers, campaign.FlowPathLanderRef{
				LanderID: landerIDs[lander.Ref],
				Weight:   lander.Weight,
			})
		}
		for _, offer := range path.Offers {
			fp.Offers = append(fp.Offers, campaign.FlowPathOfferRef{
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
	cfg campaign.IngressCostConfigDTO,
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

func ingressCostConfigDTOFromRaw(raw []byte) *campaign.IngressCostConfigDTO {
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
	return &campaign.IngressCostConfigDTO{
		Param:    param,
		Scale:    scale,
		MaxMicro: parsed.MaxMicro,
		Policy:   policy,
	}
}

func flowIDOrNil(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

func WriteCampaignImportValidationJSON(ctx context.Context, path string, spec reportjob.ReportJobSpec) error {
	kind := migrationsource.SourceKind(strings.TrimSpace(spec.ImportSourceKind))
	if kind == "" {
		return fmt.Errorf("import_source_kind required")
	}
	payload := []byte(strings.TrimSpace(string(spec.ImportPayload)))
	if len(payload) == 0 {
		return fmt.Errorf("import_payload required")
	}
	if len(payload) > migrationsource.MaxPayloadBytes {
		return fmt.Errorf("import_payload too large")
	}
	result, err := migrationsource.Preview(kind, payload, nil)
	if err != nil {
		return campaign.ErrValidationf(err.Error())
	}
	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o640)
}
