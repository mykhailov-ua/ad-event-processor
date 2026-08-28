package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/naming"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

var emailPIIPattern = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)

func (s *Service) ListAuditLogs(ctx context.Context, limit, offset int32, redactPII bool) ([]AuditLogDTO, int64, error) {
	rows, total, err := s.ListAuditLogRows(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return coldpathMapAuditRows(rows, redactPII), total, nil
}

func coldpathMapAuditRows(rows []db.AdminAuditLog, redactPII bool) []AuditLogDTO {
	out := make([]AuditLogDTO, len(rows))
	for i, row := range rows {
		changes := row.Changes
		metadata := row.Metadata
		if redactPII {
			changes = redactJSONPII(changes)
			metadata = redactJSONPII(metadata)
		}
		dto := AuditLogDTO{
			ID:         row.ID,
			Action:     row.Action,
			TargetType: row.TargetType,
			Changes:    changes,
			Metadata:   metadata,
			IsMasked:   row.IsMasked,
		}
		if row.AdminID.Valid {
			dto.AdminID = uuid.UUID(row.AdminID.Bytes).String()
		}
		if row.TargetID.Valid {
			dto.TargetID = uuid.UUID(row.TargetID.Bytes).String()
		}
		if row.CreatedAt.Valid {
			dto.CreatedAt = row.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z")
		}
		out[i] = dto
	}
	return out
}

func redactJSONPII(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return []byte(redactStringPII(string(raw)))
	}
	redactValuePII(&v)
	out, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	return out
}

func redactValuePII(v *any) {
	switch val := (*v).(type) {
	case map[string]any:
		for k, child := range val {
			lk := strings.ToLower(k)
			switch {
			case lk == "email" || strings.Contains(lk, "email"):
				val[k] = "[REDACTED_EMAIL]"
			case lk == "ip" || lk == "ip_address" || strings.HasSuffix(lk, "_ip"):
				val[k] = "[REDACTED_IP]"
			default:
				childCopy := child
				redactValuePII(&childCopy)
				val[k] = childCopy
			}
		}
	case []any:
		for i := range val {
			redactValuePII(&val[i])
		}
	case string:
		*v = redactStringPII(val)
	}
}

func redactStringPII(s string) string {
	if net.ParseIP(s) != nil {
		return "[REDACTED_IP]"
	}
	if emailPIIPattern.MatchString(s) {
		return emailPIIPattern.ReplaceAllString(s, "[REDACTED_EMAIL]")
	}
	return s
}

func (s *Service) RetryNotification(ctx context.Context, notificationID string) error {
	id, err := uuid.Parse(notificationID)
	if err != nil {
		return fmt.Errorf("invalid notification id: %w", err)
	}
	tag, err := s.GetPool().Exec(ctx, `
		UPDATE notify.notifications
		SET status = 'PENDING',
		 retry_count = 0,
		 error_message = NULL,
		 claimed_at = NULL,
		 updated_at = now()
		WHERE id = $1 AND status = 'FAILED'`, id)
	if err != nil {
		return fmt.Errorf("retry notification: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.Join(pgx.ErrNoRows, fmt.Errorf("notification not in FAILED state"))
	}
	return nil
}

func (s *Service) WarmCampaignBudget(ctx context.Context, campaignID uuid.UUID) (int64, error) {
	redisClient := s.redisClientForCampaign(campaignID)
	if redisClient == nil {
		return 0, fmt.Errorf("no redis client available")
	}
	worker := NewOutboxWorker(s)
	remaining, err := worker.campaignRemainingBudget(ctx, campaignID)
	if err != nil {
		return 0, err
	}
	if remaining <= 0 {
		return 0, nil
	}
	_, err = redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		return worker.setCampaignBudgetRemaining(ctx, pipe, campaignID.String(), campaignID, 0)
	})
	if err != nil {
		return 0, err
	}
	return remaining, nil
}

func (s *Service) EnforceSelfServeCreateLimits(ctx context.Context, customerID uuid.UUID, budgetMicro int64) error {
	if s.cfg == nil {
		return nil
	}
	if s.cfg.SelfServeBudgetMinMicro > 0 && budgetMicro < s.cfg.SelfServeBudgetMinMicro {
		return fmt.Errorf("%w: minimum %d micro", ErrSelfServeBudgetOutOfRange, s.cfg.SelfServeBudgetMinMicro)
	}
	if s.cfg.SelfServeBudgetMaxMicro > 0 && budgetMicro > s.cfg.SelfServeBudgetMaxMicro {
		return fmt.Errorf("%w: maximum %d micro", ErrSelfServeBudgetOutOfRange, s.cfg.SelfServeBudgetMaxMicro)
	}

	var active int64
	err := s.GetPool().QueryRow(ctx, `
		SELECT COUNT(*) FROM campaigns
		WHERE customer_id = $1 AND status = 'ACTIVE'`, customerID).Scan(&active)
	if err != nil {
		return fmt.Errorf("count active campaigns: %w", err)
	}
	if s.cfg.SelfServeMaxActiveCampaigns > 0 && int(active) >= s.cfg.SelfServeMaxActiveCampaigns {
		return ErrSelfServeActiveCampaignLimit
	}

	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)
	var createdToday int64
	err = s.GetPool().QueryRow(ctx, `
		SELECT COUNT(*) FROM campaigns
		WHERE customer_id = $1 AND created_at >= $2`, customerID, startOfDay).Scan(&createdToday)
	if err != nil {
		return fmt.Errorf("count daily campaign creates: %w", err)
	}
	if s.cfg.SelfServeMaxCreatesPerDay > 0 && int(createdToday) >= s.cfg.SelfServeMaxCreatesPerDay {
		return ErrSelfServeDailyCreateLimit
	}
	return nil
}

var (
	ErrFeedbackInvalidType  = errors.New("invalid feedback type")
	ErrFeedbackInvalidEmail = errors.New("invalid contact email")
	ErrFeedbackEmptyMessage = errors.New("message is required")
)

func (s *Service) SupportFeedbackMeta(ctx context.Context) (SupportFeedbackMeta, error) {
	meta := SupportFeedbackMeta{
		BinaryVersion: os.Getenv(naming.LegacyVendorEnvKey("BINARY_VERSION")),
	}
	if meta.BinaryVersion == "" {
		meta.BinaryVersion = "dev"
	}
	if s == nil || s.GetPool() == nil {
		return meta, nil
	}
	var deploymentID uuid.UUID
	err := s.GetPool().QueryRow(ctx, `
		SELECT deployment_id
		FROM billing.license_status
		LIMIT 1`).Scan(&deploymentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return meta, nil
		}
		return meta, err
	}
	if deploymentID != uuid.Nil {
		meta.DeploymentID = deploymentID.String()
	}
	return meta, nil
}

func (s *Service) RecordSupportFeedback(ctx context.Context, in SupportFeedbackRecord) (uuid.UUID, error) {
	feedbackType := strings.ToLower(strings.TrimSpace(in.Type))
	if feedbackType != "bug" && feedbackType != "feature" && feedbackType != "support" {
		return uuid.Nil, ErrFeedbackInvalidType
	}
	email := strings.TrimSpace(in.ContactEmail)
	if email == "" || !strings.Contains(email, "@") {
		return uuid.Nil, ErrFeedbackInvalidEmail
	}
	message := strings.TrimSpace(in.Message)
	if message == "" {
		return uuid.Nil, ErrFeedbackEmptyMessage
	}
	if len(message) > 8000 {
		return uuid.Nil, errValidation("message exceeds 8000 characters")
	}
	if in.AttachBundle && len(in.BundleGzip) == 0 {
		return uuid.Nil, errValidation("bundle attachment required when attach_bundle is true")
	}
	if !in.AttachBundle && len(in.BundleGzip) > 0 {
		in.BundleGzip = nil
	}
	if s == nil || s.GetPool() == nil {
		return uuid.Nil, fmt.Errorf("service unavailable")
	}

	id := uuid.New()
	var submitter pgtype.UUID
	if in.SubmitterID != uuid.Nil {
		submitter = domain.ToUUID(in.SubmitterID)
	}
	err := db.New(s.GetPool()).InsertSupportFeedback(ctx, db.InsertSupportFeedbackParams{
		ID:            domain.ToUUID(id),
		FeedbackType:  feedbackType,
		ContactEmail:  email,
		Message:       message,
		DeploymentID:  in.DeploymentID,
		BinaryVersion: in.BinaryVersion,
		Sku:           "",
		AttachBundle:  in.AttachBundle,
		BundleGzip:    in.BundleGzip,
		SubmitterID:   submitter,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert support feedback: %w", err)
	}
	return id, nil
}
