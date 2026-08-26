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

const (
	campaignExportVersion   = 1
	maxCampaignImportBytes  = 64 * 1024
	defaultImportNameSuffix = " (imported)"
)

type CampaignExportBundle struct {
	ExportVersion               int                     `json:"export_version"`
	ExportedAt                  string                  `json:"exported_at"`
	Campaign                    CampaignExportCampaign  `json:"campaign"`
	Flow                        *CampaignExportFlow     `json:"flow,omitempty"`
	Landers                     []CampaignExportLander  `json:"landers,omitempty"`
	Offers                      []CampaignExportOffer   `json:"offers,omitempty"`
	PostbackConfig              *CampaignExportPostback `json:"postback_config,omitempty"`
	ConversionMappings          []ConversionMappingDTO  `json:"conversion_mappings,omitempty"`
	IntegrationSchemaName       string                  `json:"integration_schema_name,omitempty"`
	StatusIntegrationSchemaName string                  `json:"status_integration_schema_name,omitempty"`
}

type CampaignExportCampaign struct {
	Name                       string                `json:"name"`
	BudgetLimitMicro           int64                 `json:"budget_limit_micro"`
	PacingMode                 string                `json:"pacing_mode,omitempty"`
	DailyBudgetMicro           int64                 `json:"daily_budget_micro,omitempty"`
	Timezone                   string                `json:"timezone,omitempty"`
	FreqLimit                  int32                 `json:"freq_limit,omitempty"`
	FreqWindow                 int32                 `json:"freq_window,omitempty"`
	TargetCountries            []string              `json:"target_countries,omitempty"`
	TargetURL                  string                `json:"target_url,omitempty"`
	SafePageURL                string                `json:"safe_page_url,omitempty"`
	SafePageEnabled            bool                  `json:"safe_page_enabled,omitempty"`
	AttestationEnabled         bool                  `json:"attestation_enabled,omitempty"`
	AttestationMode            string                `json:"attestation_mode,omitempty"`
	AttestationTTLSec          int32                 `json:"attestation_ttl_sec,omitempty"`
	DmrEnabled                 bool                  `json:"dmr_enabled,omitempty"`
	CIDRBlockEnabled           bool                  `json:"cidr_block_enabled,omitempty"`
	ProxyVPNBlockEnabled       bool                  `json:"proxy_vpn_block_enabled,omitempty"`
	ModeratorIntelEnabled      bool                  `json:"moderator_intel_enabled,omitempty"`
	ReviewTrafficAction        string                `json:"review_traffic_action,omitempty"`
	TLSFingerprintBlockEnabled bool                  `json:"tls_fingerprint_block_enabled,omitempty"`
	ConnTypePolicy             string                `json:"conn_type_policy,omitempty"`
	LinkSigningEnabled         bool                  `json:"link_signing_enabled,omitempty"`
	LinkSigningTTLSec          int32                 `json:"link_signing_ttl_sec,omitempty"`
	ClickDelivery              string                `json:"click_delivery,omitempty"`
	ProxyUpstreamURL           string                `json:"proxy_upstream_url,omitempty"`
	ProxyRewriteAssets         bool                  `json:"proxy_rewrite_assets,omitempty"`
	ReferrerFilter             string                `json:"referrer_filter,omitempty"`
	StartAt                    string                `json:"start_at,omitempty"`
	EndAt                      string                `json:"end_at,omitempty"`
	DaypartHours               []int16               `json:"daypart_hours,omitempty"`
	IngressCostConfig          *IngressCostConfigDTO `json:"ingress_cost_config,omitempty"`
	TrafficTemplateID          string                `json:"traffic_template_id,omitempty"`
	ClickQueryParams           map[string]string     `json:"click_query_params,omitempty"`
	CreativePayload            json.RawMessage       `json:"creative_payload,omitempty"`
	FraudThresholdPass         int16                 `json:"fraud_threshold_pass,omitempty"`
	FraudThresholdSuspect      int16                 `json:"fraud_threshold_suspect,omitempty"`
	FraudThresholdIVT          int16                 `json:"fraud_threshold_ivt,omitempty"`
	FraudThresholdBlock        int16                 `json:"fraud_threshold_block,omitempty"`
	SilentRejectEnabled        bool                  `json:"silent_reject_enabled,omitempty"`
}

type CampaignExportLander struct {
	Ref  string `json:"ref"`
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

type CampaignExportOffer struct {
	Ref  string `json:"ref"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type CampaignExportFlow struct {
	Name  string                   `json:"name"`
	Paths []CampaignExportFlowPath `json:"paths"`
}

type CampaignExportFlowPath struct {
	Weight  int32                         `json:"weight"`
	Filters *FlowPathFiltersDTO           `json:"filters,omitempty"`
	Landers []CampaignExportFlowLanderRef `json:"landers"`
	Offers  []CampaignExportFlowOfferRef  `json:"offers"`
}

type CampaignExportFlowLanderRef struct {
	Ref    string `json:"ref"`
	Weight int32  `json:"weight"`
}

type CampaignExportFlowOfferRef struct {
	Ref      string `json:"ref"`
	Weight   int32  `json:"weight"`
	CapDaily *int32 `json:"cap_daily,omitempty"`
	CapTotal *int32 `json:"cap_total,omitempty"`
}

type CampaignExportPostback struct {
	Provider      string `json:"provider"`
	URLTemplate   string `json:"url_template"`
	TargetEvent   string `json:"target_event,omitempty"`
	TestEventCode string `json:"test_event_code,omitempty"`
}

type ImportCampaignSpec struct {
	CustomerID     uuid.UUID
	NameOverride   string
	BudgetOverride *int64
	IdempotencyKey string
	Bundle         CampaignExportBundle
}

type ImportCampaignResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *Service) ExportCampaign(ctx context.Context, campaignID uuid.UUID) (CampaignExportBundle, error) {
	if s == nil || s.pool == nil {
		return CampaignExportBundle{}, fmt.Errorf("service unavailable")
	}
	row, err := db.New(s.pool).GetCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return CampaignExportBundle{}, mapNotFound(err, ErrCampaignNotFound)
	}
	if err := assertMediaBuyerCampaignAccess(ctx, row); err != nil {
		return CampaignExportBundle{}, err
	}
	if row.DeletedAt.Valid {
		return CampaignExportBundle{}, ErrCampaignNotFound
	}

	bundle := CampaignExportBundle{
		ExportVersion: campaignExportVersion,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		Campaign:      campaignRowToExport(row),
	}

	if row.FlowID.Valid {
		flowID := uuid.UUID(row.FlowID.Bytes)
		flow, flowErr := s.GetFlow(ctx, flowID)
		if flowErr != nil {
			return CampaignExportBundle{}, flowErr
		}
		exportFlow, landerRefs, offerRefs, convErr := exportFlowBundle(flow)
		if convErr != nil {
			return CampaignExportBundle{}, convErr
		}
		bundle.Flow = exportFlow
		if err := s.enrichExportFlowAssets(ctx, &bundle, landerRefs, offerRefs); err != nil {
			return CampaignExportBundle{}, err
		}
	}

	q := db.New(s.pool)
	if pb, err := q.GetPostbackConfig(ctx, domain.ToUUID(campaignID)); err == nil {
		bundle.PostbackConfig = &CampaignExportPostback{
			Provider:      pb.Provider,
			URLTemplate:   pb.UrlTemplate,
			TargetEvent:   pb.TargetEvent,
			TestEventCode: pb.TestEventCode,
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return CampaignExportBundle{}, err
	}

	mappings, err := q.ListConversionMappingsByCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return CampaignExportBundle{}, err
	}
	for i := range mappings {
		bundle.ConversionMappings = append(bundle.ConversionMappings, conversionMappingToDTO(&mappings[i]))
	}

	if row.IntegrationSchemaID.Valid {
		bundle.IntegrationSchemaName, _ = s.integrationSchemaName(ctx, uuid.UUID(row.IntegrationSchemaID.Bytes))
	}
	if row.StatusIntegrationSchemaID.Valid {
		bundle.StatusIntegrationSchemaName, _ = s.integrationSchemaName(ctx, uuid.UUID(row.StatusIntegrationSchemaID.Bytes))
	}

	raw, err := json.Marshal(bundle)
	if err != nil {
		return CampaignExportBundle{}, err
	}
	if len(raw) > maxCampaignImportBytes {
		return CampaignExportBundle{}, fmt.Errorf("export bundle exceeds %d bytes", maxCampaignImportBytes)
	}
	return bundle, nil
}

func (s *Service) ImportCampaign(ctx context.Context, spec ImportCampaignSpec) (ImportCampaignResult, error) {
	if s == nil || s.pool == nil {
		return ImportCampaignResult{}, fmt.Errorf("service unavailable")
	}
	if spec.CustomerID == uuid.Nil {
		return ImportCampaignResult{}, errValidation("customer_id is required")
	}
	if strings.TrimSpace(spec.IdempotencyKey) == "" {
		return ImportCampaignResult{}, errValidation("idempotency key is required")
	}
	if spec.Bundle.ExportVersion != campaignExportVersion {
		return ImportCampaignResult{}, errValidation("unsupported export_version")
	}
	raw, err := json.Marshal(spec.Bundle)
	if err != nil {
		return ImportCampaignResult{}, err
	}
	if len(raw) > maxCampaignImportBytes {
		return ImportCampaignResult{}, errValidation(fmt.Sprintf("import bundle exceeds %d bytes", maxCampaignImportBytes))
	}

	camp := spec.Bundle.Campaign
	name := strings.TrimSpace(camp.Name)
	if spec.NameOverride != "" {
		name = strings.TrimSpace(spec.NameOverride)
	}
	if name == "" {
		return ImportCampaignResult{}, errValidation("campaign name is required")
	}
	budget := camp.BudgetLimitMicro
	if spec.BudgetOverride != nil {
		budget = *spec.BudgetOverride
	}
	if budget <= 0 {
		return ImportCampaignResult{}, errValidation("budget_limit_micro must be positive")
	}

	startAt, endAt, err := parseExportSchedule(camp.StartAt, camp.EndAt)
	if err != nil {
		return ImportCampaignResult{}, errValidation(err.Error())
	}
	if err := validateDaypartHours(camp.DaypartHours); err != nil {
		return ImportCampaignResult{}, errValidation(err.Error())
	}
	if err := validateSchedule(startAt, endAt); err != nil {
		return ImportCampaignResult{}, errValidation(err.Error())
	}

	var (
		newCampaignID uuid.UUID
		importedName  string
		flowImported  bool
	)
	err = pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
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
			return fmt.Errorf("%w ledger row for key %q", ErrIncompleteIdempotency, spec.IdempotencyKey)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("idempotency lookup failed: %w", err)
		}

		cust, err := q.GetCustomerForUpdate(ctx, domain.ToUUID(spec.CustomerID))
		if err != nil {
			return mapNotFound(err, ErrCustomerNotFound)
		}
		if cust.Balance+cust.AllowedOverdraft < budget {
			return ErrInsufficientBalance
		}

		newCampaignID, err = uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate campaign id: %w", err)
		}
		importedName = name

		landerIDs, offerIDs, flowPaths, err := s.resolveImportFlowRefs(ctx, tx, spec.Bundle)
		if err != nil {
			return err
		}
		if len(flowPaths) > 0 {
			if err := s.validateFlowPathsWithIDs(ctx, tx, flowPaths, landerIDs, offerIDs); err != nil {
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
		initialStatus := resolveScheduleStatus(time.Now(), startAt, endAt)
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
			Timezone:        defaultTimezone(camp.Timezone),
			FreqLimit:       pgtype.Int4{Int32: camp.FreqLimit, Valid: camp.FreqLimit > 0},
			FreqWindow:      pgtype.Int4{Int32: camp.FreqWindow, Valid: camp.FreqWindow > 0},
			TargetCountries: countriesOrEmpty(camp.TargetCountries),
			BrandFcapKey:    brandFcapKey,
			StartAt:         toTimestamptz(startAt),
			EndAt:           toTimestamptz(endAt),
			DaypartHours:    daypartOrEmpty(camp.DaypartHours),
		}); err != nil {
			return err
		}

		if _, err = q.UpdateCampaignAdmin(ctx, db.UpdateCampaignAdminParams{
			ID:                         domain.ToUUID(newCampaignID),
			Name:                       importedName,
			DailyBudget:                camp.DailyBudgetMicro,
			Timezone:                   defaultTimezone(camp.Timezone),
			FreqLimit:                  pgtype.Int4{Int32: camp.FreqLimit, Valid: camp.FreqLimit > 0},
			FreqWindow:                 pgtype.Int4{Int32: camp.FreqWindow, Valid: camp.FreqWindow > 0},
			TargetCountries:            countriesOrEmpty(camp.TargetCountries),
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

		integrationID, err := s.resolveIntegrationSchemaID(ctx, tx, spec.Bundle.IntegrationSchemaName)
		if err != nil {
			return err
		}
		statusIntegrationID, err := s.resolveIntegrationSchemaID(ctx, tx, spec.Bundle.StatusIntegrationSchemaName)
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

		if err := applyCampaignClickPresetTx(ctx, tx, newCampaignID, camp.TrafficTemplateID, camp.ClickQueryParams); err != nil {
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

		normalized, err := normalizeConversionMappings(spec.Bundle.ConversionMappings)
		if err != nil {
			return errValidation(err.Error())
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

		s.AuditLog(ctx, q, uuid.Nil, "IMPORT_CAMPAIGN", "campaign", &newCampaignID, auditImportCampaignChange{
			Name: importedName,
		}, auditIdempotencyMeta{IdempotencyKey: spec.IdempotencyKey})

		return s.emitCampaignLifecycleOutbox(ctx, q, newCampaignID, initialStatus, budget)
	})
	if err != nil {
		return ImportCampaignResult{}, err
	}

	_ = s.publishCampaignUpdate(ctx, newCampaignID.String())
	if flowImported {
		_ = s.publishFlowReload(ctx)
	}

	return ImportCampaignResult{ID: newCampaignID.String(), Name: importedName}, nil
}

type auditImportCampaignChange struct {
	Name string `json:"name"`
}

func campaignRowToExport(row db.Campaign) CampaignExportCampaign {
	out := CampaignExportCampaign{
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
	out.TrafficTemplateID = formatOptionalText(row.TrafficTemplateID)
	out.ClickQueryParams = clickQueryParamsFromRaw(row.ClickQueryParams)
	return out
}

func exportFlowBundle(flow FlowDTO) (*CampaignExportFlow, map[uuid.UUID]CampaignExportLander, map[uuid.UUID]CampaignExportOffer, error) {
	var paths []FlowPathDTO
	if len(flow.Paths) > 0 {
		if err := json.Unmarshal(flow.Paths, &paths); err != nil {
			return nil, nil, nil, fmt.Errorf("parse flow paths: %w", err)
		}
	}
	landerByID := make(map[uuid.UUID]CampaignExportLander)
	offerByID := make(map[uuid.UUID]CampaignExportOffer)
	exportPaths := make([]CampaignExportFlowPath, 0, len(paths))
	for _, path := range paths {
		ep := CampaignExportFlowPath{Weight: path.Weight, Filters: path.Filters}
		for _, lander := range path.Landers {
			ref := lander.LanderID.String()
			landerByID[lander.LanderID] = CampaignExportLander{Ref: ref}
			ep.Landers = append(ep.Landers, CampaignExportFlowLanderRef{Ref: ref, Weight: lander.Weight})
		}
		for _, offer := range path.Offers {
			ref := offer.OfferID.String()
			offerByID[offer.OfferID] = CampaignExportOffer{Ref: ref}
			ep.Offers = append(ep.Offers, CampaignExportFlowOfferRef{
				Ref:      ref,
				Weight:   offer.Weight,
				CapDaily: offer.CapDaily,
				CapTotal: offer.CapTotal,
			})
		}
		exportPaths = append(exportPaths, ep)
	}
	return &CampaignExportFlow{Name: flow.Name, Paths: exportPaths}, landerByID, offerByID, nil
}

func (s *Service) enrichExportFlowAssets(ctx context.Context, bundle *CampaignExportBundle, landerRefs map[uuid.UUID]CampaignExportLander, offerRefs map[uuid.UUID]CampaignExportOffer) error {
	if bundle == nil || s.pool == nil {
		return nil
	}
	landerIDs := make([]uuid.UUID, 0, len(landerRefs))
	for id := range landerRefs {
		landerIDs = append(landerIDs, id)
	}
	if len(landerIDs) > 0 {
		rows, err := s.pool.Query(ctx, `SELECT id, name, COALESCE(url, '') FROM landers WHERE id = ANY($1)`, landerIDs)
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
		rows, err := s.pool.Query(ctx, `SELECT id, name, url FROM offers WHERE id = ANY($1)`, offerIDs)
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
	bundle.Landers = make([]CampaignExportLander, 0, len(landerRefs))
	for _, row := range landerRefs {
		bundle.Landers = append(bundle.Landers, row)
	}
	bundle.Offers = make([]CampaignExportOffer, 0, len(offerRefs))
	for _, row := range offerRefs {
		bundle.Offers = append(bundle.Offers, row)
	}
	return nil
}
