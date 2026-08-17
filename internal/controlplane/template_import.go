package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/integrationschema"
	"github.com/bidshard/ad-event-processor/internal/postback"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) ListBundledTemplates(_ context.Context) []integrationschema.TemplateCatalogEntry {
	return integrationschema.GMM4TemplateCatalog
}

func (s *Service) ImportBundledTemplates(ctx context.Context, names []string) ([]adminapi.IntegrationSchemaDTO, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	wantAll := len(names) == 0
	want := make(map[string]struct{}, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n != "" {
			want[n] = struct{}{}
		}
	}
	var out []adminapi.IntegrationSchemaDTO
	for _, entry := range integrationschema.GMM4TemplateCatalog {
		if !wantAll {
			if _, ok := want[entry.Name]; !ok {
				continue
			}
		}
		dto, err := s.importBundledTemplate(ctx, entry)
		if err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, nil
}

func (s *Service) importBundledTemplate(ctx context.Context, entry integrationschema.TemplateCatalogEntry) (adminapi.IntegrationSchemaDTO, error) {
	raw, kind, parsed, err := integrationschema.LoadBundledTemplate(entry)
	if err != nil {
		return adminapi.IntegrationSchemaDTO{}, err
	}
	jsonBody, err := json.Marshal(parsed)
	if err != nil {
		return adminapi.IntegrationSchemaDTO{}, err
	}
	_ = raw
	var id uuid.UUID
	err = s.pool.QueryRow(ctx, `
		INSERT INTO integration_schemas (name, version, kind, body)
		VALUES ($1, $2, $3, $4::jsonb)
		ON CONFLICT (name, version) DO UPDATE
		SET kind = EXCLUDED.kind, body = EXCLUDED.body, updated_at = NOW()
		RETURNING id`,
		entry.Name, entry.Version, string(kind), jsonBody,
	).Scan(&id)
	if err != nil {
		return adminapi.IntegrationSchemaDTO{}, err
	}
	return s.getIntegrationSchemaDTO(ctx, id)
}

func (s *Service) getIntegrationSchemaDTO(ctx context.Context, id uuid.UUID) (adminapi.IntegrationSchemaDTO, error) {
	var dto adminapi.IntegrationSchemaDTO
	var kind string
	var body []byte
	var created, updated time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, version, kind, body, created_at, updated_at
		FROM integration_schemas WHERE id = $1`, id).Scan(
		&id, &dto.Name, &dto.Version, &kind, &body, &created, &updated,
	)
	if err != nil {
		return adminapi.IntegrationSchemaDTO{}, err
	}
	dto.ID = id.String()
	dto.Kind = kind
	dto.Schema = json.RawMessage(body)
	dto.CreatedAt = created
	dto.UpdatedAt = updated
	return dto, nil
}

func (s *Service) resolveSchemaIDByName(ctx context.Context, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT id FROM integration_schemas WHERE name = $1 ORDER BY version DESC LIMIT 1`, name,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("schema %q not imported", name)
		}
		return uuid.Nil, err
	}
	return id, nil
}

func (s *Service) ApplyCampaignTemplates(ctx context.Context, campaignID uuid.UUID, req adminapi.ApplyCampaignTemplatesRequest) (adminapi.ApplyCampaignTemplatesResult, error) {
	if s == nil || s.pool == nil {
		return adminapi.ApplyCampaignTemplatesResult{}, fmt.Errorf("service unavailable")
	}
	if campaignID == uuid.Nil {
		return adminapi.ApplyCampaignTemplatesResult{}, fmt.Errorf("campaign id required")
	}
	result := adminapi.ApplyCampaignTemplatesResult{CampaignID: campaignID.String()}

	trackingDomain := strings.TrimSpace(req.TrackingDomain)
	if trackingDomain == "" {
		if cfg, _, err := s.GetPlatformConfig(ctx); err == nil {
			trackingDomain = cfg.TrackingDomain
		}
	}

	if src := strings.TrimSpace(req.TrafficSource); src != "" {
		schemaID, err := s.resolveSchemaIDByName(ctx, src)
		if err != nil {
			return result, err
		}
		applied, err := s.applyIntegrationSchema(ctx, campaignID, schemaID, trackingDomain)
		if err != nil {
			return result, err
		}
		result.TrafficSource = applied
	}

	if net := strings.TrimSpace(req.AffiliateNetwork); net != "" {
		outID, err := s.resolveSchemaIDByName(ctx, net)
		if err != nil {
			return result, err
		}
		applied, err := s.applyIntegrationSchema(ctx, campaignID, outID, trackingDomain)
		if err != nil {
			return result, err
		}
		result.AffiliatePostback = applied
		if statusName, ok := integrationschema.AffiliateStatusTemplateName(net); ok {
			statusID, err := s.resolveSchemaIDByName(ctx, statusName)
			if err != nil {
				return result, err
			}
			statusApplied, err := s.applyIntegrationSchema(ctx, campaignID, statusID, trackingDomain)
			if err != nil {
				return result, err
			}
			result.AffiliateStatus = statusApplied
		}
	}

	return result, nil
}

func (s *Service) applyIntegrationSchema(ctx context.Context, campaignID, schemaID uuid.UUID, trackingDomain string) (map[string]string, error) {
	var kind string
	var schemaBody []byte
	err := s.pool.QueryRow(ctx, `SELECT kind, body FROM integration_schemas WHERE id = $1`, schemaID).Scan(&kind, &schemaBody)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("schema not found")
		}
		return nil, err
	}

	q := db.New(s.pool)
	if _, err := q.GetCampaign(ctx, pgtype.UUID{Bytes: campaignID, Valid: true}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("campaign not found")
		}
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	applied := map[string]string{"schema_id": schemaID.String(), "kind": kind}
	switch integrationschema.Kind(kind) {
	case integrationschema.KindInboundTokens:
		parsedKind, parsed, err := integrationschema.ParseDocument(schemaBody)
		if err != nil || parsedKind != integrationschema.KindInboundTokens {
			return nil, fmt.Errorf("invalid inbound schema")
		}
		inbound := parsed.(*integrationschema.InboundTokensSchema)
		trackingURL := integrationschema.BuildInboundTrackingURL(trackingDomain, inbound)
		if _, err := tx.Exec(ctx, `
			UPDATE campaigns
			SET integration_schema_id = $2, target_url = $3, updated_at = NOW()
			WHERE id = $1`, campaignID, schemaID, trackingURL); err != nil {
			return nil, err
		}
		applied["target_url"] = trackingURL
	case integrationschema.KindOutboundPostback:
		tpl, err := integrationschema.OutboundURLTemplateFromBody(schemaBody)
		if err != nil {
			return nil, err
		}
		key := postbackEncryptionKey(s)
		encToken, err := postback.EncryptAESGCM([]byte("integration-schema"), key)
		if err != nil {
			return nil, fmt.Errorf("encryption failed")
		}
		txQ := db.New(tx)
		if err := txQ.UpsertPostbackConfig(ctx, db.UpsertPostbackConfigParams{
			CampaignID:        pgtype.UUID{Bytes: campaignID, Valid: true},
			Provider:          "webhook",
			UrlTemplate:       tpl,
			ApiTokenEncrypted: encToken,
			TargetEvent:       "conversion",
		}); err != nil {
			return nil, err
		}
		applied["url_template"] = tpl
	case integrationschema.KindStatusMapping:
		if _, err := tx.Exec(ctx, `
			UPDATE campaigns SET status_integration_schema_id = $2, updated_at = NOW() WHERE id = $1`,
			campaignID, schemaID); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported schema kind %q", kind)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	_ = s.publishCampaignUpdate(ctx, campaignID.String())
	return applied, nil
}

func postbackEncryptionKey(s *Service) []byte {
	_ = s
	return []byte("postback-encryption-secret-key32")
}

func (s *Service) ImportGMM4Templates(ctx context.Context) ([]adminapi.IntegrationSchemaDTO, error) {
	return s.ImportBundledTemplates(ctx, nil)
}
