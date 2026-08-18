package controlplane

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const fraudIntegrationsMaxCampaigns = 50

func postbackConfigConfigured(provider, urlTemplate string, apiTokenLen int) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" || strings.TrimSpace(urlTemplate) == "" {
		return false
	}
	if provider == "webhook" {
		return true
	}
	switch provider {
	case "facebook", "google", "tiktok":
		return apiTokenLen > 0
	default:
		return false
	}
}

func deriveFraudIntegrationHealth(configured bool, dlqCount int64, lastSuccess *time.Time) string {
	if !configured {
		return "unconfigured"
	}
	if dlqCount > 0 {
		return "failing"
	}
	if lastSuccess == nil {
		return "idle"
	}
	return "configured"
}

func (s *Service) ListFraudIntegrationsForCustomer(ctx context.Context, customerID uuid.UUID) ([]FraudIntegrationDTO, error) {
	if customerID == uuid.Nil {
		return nil, errValidation("customer_id is required")
	}
	if s == nil || s.GetPool() == nil {
		return nil, fmt.Errorf("postgres pool not configured")
	}

	rows, err := s.GetPool().Query(ctx, `
		SELECT
			c.id,
			c.name,
			COALESCE(p.provider, '') AS provider,
			COALESCE(p.url_template, '') AS url_template,
			COALESCE(length(p.api_token_encrypted), 0) AS api_token_len,
			d.last_success_at,
			COALESCE(q.dlq_count, 0) AS dlq_count,
			COALESCE(le.last_error, '') AS last_error
		FROM campaigns c
		LEFT JOIN postback_configs p ON p.campaign_id = c.id
		LEFT JOIN (
			SELECT campaign_id, MAX(created_at) AS last_success_at
			FROM postback_dispatches
			WHERE status IN ('SENT', 'DELIVERED')
			GROUP BY campaign_id
		) d ON d.campaign_id = c.id
		LEFT JOIN (
			SELECT campaign_id, COUNT(*)::bigint AS dlq_count
			FROM postback_dlq
			WHERE status = 'FAILED'
			GROUP BY campaign_id
		) q ON q.campaign_id = c.id
		LEFT JOIN LATERAL (
			SELECT last_error
			FROM postback_dlq
			WHERE campaign_id = c.id AND status = 'FAILED'
			ORDER BY created_at DESC
			LIMIT 1
		) le ON true
		WHERE c.customer_id = $1 AND c.deleted_at IS NULL
		ORDER BY c.name ASC
		LIMIT $2`, domain.ToUUID(customerID), fraudIntegrationsMaxCampaigns)
	if err != nil {
		return nil, fmt.Errorf("query fraud integrations: %w", err)
	}
	defer rows.Close()

	out := make([]FraudIntegrationDTO, 0, fraudIntegrationsMaxCampaigns)
	for rows.Next() {
		var campaignID pgtype.UUID
		var name, provider, urlTemplate, lastError string
		var apiTokenLen int
		var dlqCount int64
		var lastSuccessAt *time.Time
		if err := rows.Scan(&campaignID, &name, &provider, &urlTemplate, &apiTokenLen, &lastSuccessAt, &dlqCount, &lastError); err != nil {
			return nil, err
		}
		configured := postbackConfigConfigured(provider, urlTemplate, apiTokenLen)
		dto := FraudIntegrationDTO{
			CampaignID:   uuid.UUID(campaignID.Bytes).String(),
			Name:         name,
			Provider:     provider,
			Configured:   configured,
			HealthStatus: deriveFraudIntegrationHealth(configured, dlqCount, lastSuccessAt),
			DLQCount:     dlqCount,
			LastError:    strings.TrimSpace(lastError),
		}
		if lastSuccessAt != nil && !lastSuccessAt.IsZero() {
			dto.LastSuccessAt = lastSuccessAt.UTC().Format(time.RFC3339)
		}
		out = append(out, dto)
	}
	return out, rows.Err()
}
