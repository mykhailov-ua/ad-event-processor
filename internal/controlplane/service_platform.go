package controlplane

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"espx/internal/domain"
	"espx/internal/domain/db"
	"espx/internal/edge/allowlist"
	"espx/internal/ledger"
	"espx/pkg/coldpath"
	"fmt"
	"math"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

var emailPIIPattern = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)

type AuditLogDTO struct {
	ID         int64           `json:"id"`
	AdminID    string          `json:"admin_id,omitempty"`
	Action     string          `json:"action"`
	TargetType string          `json:"target_type"`
	TargetID   string          `json:"target_id,omitempty"`
	Changes    json.RawMessage `json:"changes"`
	Metadata   json.RawMessage `json:"metadata"`
	IsMasked   bool            `json:"is_masked"`
	CreatedAt  string          `json:"created_at"`
}

func (s *Service) ListAuditLogsRedacted(ctx context.Context, limit, offset int32, redactPII bool) ([]AuditLogDTO, int64, error) {
	rows, total, err := s.ListAuditLogs(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]AuditLogDTO, len(rows))
	for i, row := range rows {
		out[i] = auditRowToDTO(row, redactPII)
	}
	return out, total, nil
}

func auditRowToDTO(row db.AdminAuditLog, redactPII bool) AuditLogDTO {
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
	return dto
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

var (
	ErrConsentInvalidSignature = errors.New("invalid consent signature")
	ErrConsentInvalidPayload   = errors.New("invalid consent payload")
)

type ConsentRecordInput struct {
	UserID    string `json:"user_id"`
	Purposes  int16  `json:"purposes"`
	Source    string `json:"source"`
	Timestamp string `json:"timestamp,omitempty"`
}

func (s *Service) RecordConsent(ctx context.Context, in ConsentRecordInput) error {
	if in.UserID == "" {
		return errValidation("user_id is required")
	}
	if in.Source == "" {
		return errValidation("source is required")
	}
	if in.Purposes < 0 {
		return errValidation("purposes must be non-negative")
	}

	hash := domain.HashUserID(in.UserID)
	adStorage, analyticsStorage := domain.ConsentFlagsFromPurposes(in.Purposes)

	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if err := q.InsertConsentEvent(ctx, db.InsertConsentEventParams{
			UserIDHash: hash,
			Purposes:   in.Purposes,
			Source:     in.Source,
		}); err != nil {
			return fmt.Errorf("insert consent event: %w", err)
		}
		if err := q.UpsertUserConsentState(ctx, db.UpsertUserConsentStateParams{
			UserIDHash:       hash,
			AdStorage:        adStorage,
			AnalyticsStorage: analyticsStorage,
			Purposes:         in.Purposes,
		}); err != nil {
			return fmt.Errorf("upsert consent state: %w", err)
		}
		payload, err := coldpath.MarshalJSON(map[string]any{
			"user_id_hash": hex.EncodeToString(hash),
			"purposes":     in.Purposes,
		})
		if err != nil {
			return fmt.Errorf("marshal consent outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "SYNC_USER_CONSENT",
			Payload:   payload,
		})
		return err
	})
}

func VerifyConsentHMAC(secret []byte, body []byte, signatureHex string) error {
	if len(secret) == 0 {
		return ErrConsentInvalidSignature
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expected := mac.Sum(nil)
	got, err := hex.DecodeString(signatureHex)
	if err != nil {
		return ErrConsentInvalidSignature
	}
	if !hmac.Equal(expected, got) {
		return ErrConsentInvalidSignature
	}
	return nil
}

func (s *Service) UpdateCampaignConsentRequirements(ctx context.Context, campaignID uuid.UUID, purposes int16) error {
	if purposes < 0 {
		return errValidation("require_consent_purposes must be non-negative")
	}
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if _, err := q.UpdateCampaignConsentPurposes(ctx, db.UpdateCampaignConsentPurposesParams{
			ID:                     domain.ToUUID(campaignID),
			RequireConsentPurposes: purposes,
		}); err != nil {
			return mapNotFound(err, ErrCampaignNotFound)
		}
		payload, err := coldpath.MarshalJSON(map[string]string{"campaign_id": campaignID.String()})
		if err != nil {
			return err
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "UPDATE_CAMPAIGN_CONSENT",
			Payload:   payload,
		})
		return err
	})
}

func (s *Service) CleanupConsentEvents(ctx context.Context) error {
	if s.cfg == nil || s.cfg.ConsentRetentionMonths <= 0 {
		return nil
	}
	threshold := time.Now().AddDate(0, -s.cfg.ConsentRetentionMonths, 0)
	return db.New(s.GetPool()).CleanupConsentEventsOlderThan(ctx, pgtype.Timestamptz{Time: threshold, Valid: true})
}

func (s *Service) CreatePrivacyErasureRequest(ctx context.Context, userID string) (uuid.UUID, error) {
	if userID == "" {
		return uuid.Nil, errValidation("user_id is required")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}
	hash := domain.HashUserID(userID)
	_, err = db.New(s.GetPool()).CreatePrivacyErasureRequest(ctx, db.CreatePrivacyErasureRequestParams{
		ID:            domain.ToUUID(id),
		UserIDHash:    hash,
		SubjectUserID: userID,
	})
	return id, err
}

func (s *Service) ProcessPrivacyErasureTick(ctx context.Context) error {
	opCtx, cancel := workerContext(ctx, workerBatchTimeout)
	defer cancel()

	q := db.New(s.GetPool())
	rows, err := q.ListPrivacyErasureRequestsByStatus(opCtx, db.ListPrivacyErasureRequestsByStatusParams{
		Status: db.PrivacyErasureStatusPENDING,
		Limit:  20,
	})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := s.advanceErasurePG(opCtx, row); err != nil {
			_ = s.failErasure(opCtx, row.ID, err)
		}
	}

	rows, err = q.ListPrivacyErasureRequestsByStatus(opCtx, db.ListPrivacyErasureRequestsByStatusParams{
		Status: db.PrivacyErasureStatusPGANONYMIZED,
		Limit:  20,
	})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := s.enqueueErasureRedisPurge(opCtx, row); err != nil {
			_ = s.failErasure(opCtx, row.ID, err)
		}
	}

	rows, err = q.ListPrivacyErasureRequestsByStatus(opCtx, db.ListPrivacyErasureRequestsByStatusParams{
		Status: db.PrivacyErasureStatusREDISPURGED,
		Limit:  20,
	})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := s.advanceErasureCH(opCtx, row); err != nil {
			_ = s.failErasure(opCtx, row.ID, err)
		}
	}
	return nil
}

func (s *Service) advanceErasurePG(ctx context.Context, row db.PrivacyErasureRequest) error {
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		locked, err := q.GetPrivacyErasureRequestForUpdate(ctx, row.ID)
		if err != nil {
			return err
		}
		if locked.Status != db.PrivacyErasureStatusPENDING {
			return nil
		}
		if locked.SubjectUserID != "" {
			if err := q.AnonymizeEventsByUserID(ctx, pgtype.Text{String: locked.SubjectUserID, Valid: true}); err != nil {
				return err
			}
		}
		if err := q.AnonymizeConsentEventsByUserHash(ctx, locked.UserIDHash); err != nil {
			return err
		}
		if err := q.DeleteUserConsentState(ctx, locked.UserIDHash); err != nil {
			return err
		}
		return q.UpdatePrivacyErasureStatus(ctx, db.UpdatePrivacyErasureStatusParams{
			ID:     locked.ID,
			Status: db.PrivacyErasureStatusPGANONYMIZED,
		})
	})
}

func (s *Service) enqueueErasureRedisPurge(ctx context.Context, row db.PrivacyErasureRequest) error {
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		locked, err := q.GetPrivacyErasureRequestForUpdate(ctx, row.ID)
		if err != nil {
			return err
		}
		if locked.Status != db.PrivacyErasureStatusPGANONYMIZED {
			return nil
		}
		if locked.LastError.Valid && locked.LastError.String == "purge_enqueued" {
			return nil
		}
		payload, err := coldpath.MarshalJSON(map[string]string{
			"erasure_id":      uuid.UUID(locked.ID.Bytes).String(),
			"user_id_hash":    hex.EncodeToString(locked.UserIDHash),
			"subject_user_id": locked.SubjectUserID,
		})
		if err != nil {
			return err
		}
		if _, err := q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "PURGE_USER_DATA",
			Payload:   payload,
		}); err != nil {
			return err
		}
		return q.UpdatePrivacyErasureStatus(ctx, db.UpdatePrivacyErasureStatusParams{
			ID:        locked.ID,
			Status:    db.PrivacyErasureStatusPGANONYMIZED,
			LastError: pgtype.Text{String: "purge_enqueued", Valid: true},
		})
	})
}

func (s *Service) advanceErasureCH(ctx context.Context, row db.PrivacyErasureRequest) error {
	userID := row.SubjectUserID
	if s.chWrite != nil && userID != "" {
		query := `ALTER TABLE fraud_events DELETE WHERE user_id = ?`
		if err := s.chWrite.Exec(ctx, query, userID); err != nil {
			return s.failErasure(ctx, row.ID, err)
		}
	}
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if err := q.UpdatePrivacyErasureStatus(ctx, db.UpdatePrivacyErasureStatusParams{
			ID:     row.ID,
			Status: db.PrivacyErasureStatusCHPURGED,
		}); err != nil {
			return err
		}
		if err := q.UpdatePrivacyErasureStatus(ctx, db.UpdatePrivacyErasureStatusParams{
			ID:     row.ID,
			Status: db.PrivacyErasureStatusCOMPLETED,
		}); err != nil {
			return err
		}
		return q.ClearErasureSubjectUserID(ctx, row.ID)
	})
}

func (s *Service) failErasure(ctx context.Context, id pgtype.UUID, err error) error {
	msg := err.Error()
	return db.New(s.GetPool()).UpdatePrivacyErasureStatus(ctx, db.UpdatePrivacyErasureStatusParams{
		ID:        id,
		Status:    db.PrivacyErasureStatusFAILED,
		LastError: pgtype.Text{String: msg, Valid: true},
	})
}

func (s *Service) PurgeUserDataRedis(ctx context.Context, hashHex, subjectUserID string) error {
	if len(s.rdbs) == 0 {
		return fmt.Errorf("no redis clients")
	}
	consentKey := domain.ConsentRedisKeyPrefix + hashHex
	pattern := "*:u:" + subjectUserID
	var firstErr error
	var success int
	for _, rdb := range s.rdbs {
		if err := rdb.Del(ctx, consentKey).Err(); err != nil && err != redis.Nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		success++
		iter := rdb.Scan(ctx, 0, pattern, 200).Iterator()
		for iter.Next(ctx) {
			_ = rdb.Del(ctx, iter.Val()).Err()
		}
		_ = iter.Err()
	}
	if success == 0 && firstErr != nil {
		return firstErr
	}
	_ = publishControlChannelToAllShards(ctx, s.rdbs, s.consentUpdateChannel(), hashHex)
	return nil
}

func (s *Service) SyncUserConsentToRedis(ctx context.Context, hashHex string, purposes int16) error {
	if len(s.rdbs) == 0 {
		return fmt.Errorf("no redis clients")
	}
	val := strconv.FormatInt(int64(purposes), 10)
	key := domain.ConsentRedisKeyPrefix + hashHex
	for _, rdb := range s.rdbs {
		if err := rdb.Set(ctx, key, val, 0).Err(); err != nil {
			return err
		}
	}
	return publishControlChannelToAllShards(ctx, s.rdbs, s.consentUpdateChannel(), hashHex)
}

func (s *Service) consentUpdateChannel() string {
	if s.cfg != nil && s.cfg.ConsentUpdateChannel != "" {
		return s.cfg.ConsentUpdateChannel
	}
	return domain.ConsentDefaultUpdateChannel
}

func (s *Service) MarkErasureRedisPurgeDone(ctx context.Context, erasureID uuid.UUID, partialErr error) error {
	status := db.PrivacyErasureStatusREDISPURGED
	if partialErr != nil {
		return db.New(s.GetPool()).UpdatePrivacyErasureStatus(ctx, db.UpdatePrivacyErasureStatusParams{
			ID:        domain.ToUUID(erasureID),
			Status:    db.PrivacyErasureStatusFAILED,
			LastError: pgtype.Text{String: partialErr.Error(), Valid: true},
		})
	}
	return db.New(s.GetPool()).UpdatePrivacyErasureStatus(ctx, db.UpdatePrivacyErasureStatusParams{
		ID:     domain.ToUUID(erasureID),
		Status: status,
	})
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
	rdb := s.getRDB(campaignID)
	if rdb == nil {
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
	_, err = rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
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

type SupportFeedbackInput struct {
	Type          string
	ContactEmail  string
	Message       string
	AttachBundle  bool
	BundleGzip    []byte
	SubmitterID   uuid.UUID
	DeploymentID  string
	BinaryVersion string
	SKU           string
}

type SupportFeedbackMeta struct {
	DeploymentID  string `json:"deployment_id"`
	BinaryVersion string `json:"binary_version"`
	SKU           string `json:"sku"`
}

func (s *Service) SupportFeedbackMeta(ctx context.Context) (SupportFeedbackMeta, error) {
	meta := SupportFeedbackMeta{
		BinaryVersion: os.Getenv("ESPX_BINARY_VERSION"),
	}
	if meta.BinaryVersion == "" {
		meta.BinaryVersion = "dev"
	}
	if s == nil || s.GetPool() == nil {
		return meta, nil
	}
	var deploymentID uuid.UUID
	var planCode string
	err := s.GetPool().QueryRow(ctx, `
		SELECT deployment_id, plan_code
		FROM billing.license_status
		LIMIT 1`).Scan(&deploymentID, &planCode)
	if err != nil {
		if err == pgx.ErrNoRows {
			return meta, nil
		}
		return meta, err
	}
	if deploymentID != uuid.Nil {
		meta.DeploymentID = deploymentID.String()
	}
	meta.SKU = planCode
	return meta, nil
}

func (s *Service) RecordSupportFeedback(ctx context.Context, in SupportFeedbackInput) (uuid.UUID, error) {
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
		Sku:           in.SKU,
		AttachBundle:  in.AttachBundle,
		BundleGzip:    in.BundleGzip,
		SubmitterID:   submitter,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert support feedback: %w", err)
	}
	return id, nil
}

func (s *Service) CreateMarginGuardPolicy(ctx context.Context, p *ledger.Policy) error {
	thresholdBps := ledger.CostOverRevenueThresholdBps(p, s.cfg)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO margin_guard_policies (campaign_id, name, min_clicks, roi_floor_pct, zero_conv_streak, cost_over_revenue_threshold_bps, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, domain.ToUUID(p.CampaignID), p.Name, p.MinClicks, p.RoiFloorPct, p.ZeroConvStreak, thresholdBps, p.IsActive)
	return err
}

func (s *Service) ListMarginGuardPolicies(ctx context.Context, campaignID uuid.UUID) ([]*ledger.Policy, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, campaign_id, name, min_clicks, roi_floor_pct, zero_conv_streak, cost_over_revenue_threshold_bps, is_active
		FROM margin_guard_policies
		WHERE campaign_id = $1
	`, domain.ToUUID(campaignID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*ledger.Policy
	for rows.Next() {
		p := &ledger.Policy{}
		if err := rows.Scan(&p.ID, &p.CampaignID, &p.Name, &p.MinClicks, &p.RoiFloorPct, &p.ZeroConvStreak, &p.CostOverRevenueThresholdBps, &p.IsActive); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, nil
}

func (s *Service) GetCampaignMargin(ctx context.Context, campaignID uuid.UUID) (map[string]any, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	windowStart := time.Now().Add(-1 * time.Hour)
	q := db.New(s.pool)
	sums, err := q.SumCampaignMarginWindow(ctx, db.SumCampaignMarginWindowParams{
		CampaignID: domain.ToUUID(campaignID),
		CreatedAt:  pgtype.Timestamp{Time: windowStart, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	thresholdBps := ledger.CostOverRevenueThresholdBps(nil, s.cfg)
	policies, err := s.ListMarginGuardPolicies(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if len(policies) > 0 {
		thresholdBps = ledger.CostOverRevenueThresholdBps(policies[0], s.cfg)
	}
	limitMicro := ledger.CostOverRevenueLimitMicro(sums.AdvertiserSpendMicro, thresholdBps)
	return map[string]any{
		"campaign_id":             campaignID.String(),
		"window_start":            windowStart.UTC().Format(time.RFC3339),
		"window_hours":            1,
		"advertiser_spend_micro":  sums.AdvertiserSpendMicro,
		"rtb_cost_micro":          sums.RtbCostMicro,
		"operator_margin_micro":   sums.OperatorMarginMicro,
		"publisher_payout_micro":  sums.PublisherPayoutMicro,
		"cost_over_revenue_limit": limitMicro,
		"threshold_bps":           thresholdBps,
		"margin_breach":           sums.RtbCostMicro > limitMicro && sums.AdvertiserSpendMicro > 0,
	}, nil
}

func (s *Service) GetMarginGuardActivity(ctx context.Context, campaignID uuid.UUID) ([]map[string]any, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, policy_id, campaign_id, placement_id, action, reason, metrics, created_at
		FROM margin_guard_activity
		WHERE campaign_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`, domain.ToUUID(campaignID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []map[string]any
	for rows.Next() {
		var id, policyID, campID uuid.UUID
		var placementID, action, reason string
		var metrics map[string]any
		var createdAt interface{}
		if err := rows.Scan(&id, &policyID, &campID, &placementID, &action, &reason, &metrics, &createdAt); err != nil {
			return nil, err
		}
		activities = append(activities, map[string]any{
			"id":           id,
			"policy_id":    policyID,
			"campaign_id":  campID,
			"placement_id": placementID,
			"action":       action,
			"reason":       reason,
			"metrics":      metrics,
			"created_at":   createdAt,
		})
	}
	return activities, nil
}

func (s *Service) RemovePlacementOverride(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	payload, err := coldpath.MarshalJSON(PausePlacementPayload{
		CampaignID:  campaignID.String(),
		PlacementID: placementID,
		Action:      "remove",
	})
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO outbox_events (event_type, payload)
		VALUES ($1, $2)`, "PAUSE_PLACEMENT", payload)
	return err
}

type mabCreativeStat struct {
	impressions int64
	clicks      int64
}

func (s *Service) optimizeBrandCreativeMABTx(ctx context.Context, tx pgx.Tx) ([]uuid.UUID, error) {
	if s.chQuery == nil {
		return nil, nil
	}
	minImps := s.cfg.MABMinImpressions
	if minImps <= 0 {
		minImps = 1000
	}
	lookbackDays := s.cfg.MABLookbackDays
	if lookbackDays <= 0 {
		lookbackDays = 90
	}

	q := db.New(tx)
	brandRows, err := q.ListDistinctBrandsWithActiveCreatives(ctx)
	if err != nil {
		return nil, fmt.Errorf("list brands for mab: %w", err)
	}
	if len(brandRows) == 0 {
		return nil, nil
	}

	lookbackEnd := time.Now().UTC()
	lookbackStart := lookbackEnd.Add(-time.Duration(lookbackDays) * 24 * time.Hour)
	chStats, err := s.queryMABCreativeStats(ctx, lookbackStart, lookbackEnd)
	if err != nil {
		return nil, err
	}

	var updatedBrands []uuid.UUID
	for _, brandRow := range brandRows {
		brandID := uuid.UUID(brandRow.Bytes)
		creatives, err := q.ListActiveBrandCreatives(ctx, brandRow)
		if err != nil {
			return nil, err
		}
		if len(creatives) < 2 {
			continue
		}

		campaignRows, err := q.ListCampaignIDsByBrand(ctx, brandRow)
		if err != nil {
			return nil, err
		}

		attributed := attributeMABStats(creatives, campaignRows, chStats, minImps)
		if !attributed.anyEligible {
			continue
		}

		newWeights := computeMABWeights(attributed.perCreative)
		brandChanged := false
		for _, cr := range creatives {
			creativeID := uuid.UUID(cr.ID.Bytes)
			newWeight, ok := newWeights[creativeID]
			if !ok || newWeight == cr.Weight {
				continue
			}
			_, err := q.UpdateBrandCreative(ctx, db.UpdateBrandCreativeParams{
				ID:         cr.ID,
				Name:       cr.Name,
				LandingUrl: cr.LandingUrl,
				Weight:     newWeight,
				Status:     cr.Status,
			})
			if err != nil {
				return nil, fmt.Errorf("update creative weight %s: %w", creativeID, err)
			}
			brandChanged = true
		}
		if brandChanged {
			updatedBrands = append(updatedBrands, brandID)
		}
	}
	return updatedBrands, nil
}

type mabAttribution struct {
	perCreative map[uuid.UUID]mabCreativeStat
	anyEligible bool
}

func attributeMABStats(
	creatives []db.BrandCreative,
	campaignRows []pgtype.UUID,
	chStats map[uuid.UUID]mabCreativeStat,
	minImps int64,
) mabAttribution {
	out := mabAttribution{perCreative: make(map[uuid.UUID]mabCreativeStat, len(creatives))}

	for creativeID, stat := range chStats {
		if stat.impressions >= minImps {
			out.perCreative[creativeID] = stat
			out.anyEligible = true
		}
	}
	if out.anyEligible {
		return out
	}

	if len(creatives) == 0 || len(campaignRows) == 0 {
		return out
	}

	var totalImps, totalClicks int64
	for _, camp := range campaignRows {
		if stat, ok := chStats[uuid.UUID(camp.Bytes)]; ok {
			totalImps += stat.impressions
			totalClicks += stat.clicks
		}
	}
	if totalImps < minImps {
		return out
	}

	shareImps := totalImps / int64(len(creatives))
	shareClicks := totalClicks / int64(len(creatives))
	if shareImps < minImps {
		return out
	}

	for _, cr := range creatives {
		creativeID := uuid.UUID(cr.ID.Bytes)
		out.perCreative[creativeID] = mabCreativeStat{
			impressions: shareImps,
			clicks:      shareClicks,
		}
	}
	out.anyEligible = true
	return out
}

func computeMABWeights(stats map[uuid.UUID]mabCreativeStat) map[uuid.UUID]int32 {
	weights := make(map[uuid.UUID]int32, len(stats))
	var sumCTR float64
	for _, stat := range stats {
		if stat.impressions > 0 {
			sumCTR += float64(stat.clicks) / float64(stat.impressions)
		}
	}
	if sumCTR <= 0 {
		for id := range stats {
			weights[id] = 1
		}
		return weights
	}
	for id, stat := range stats {
		ctr := float64(stat.clicks) / float64(stat.impressions)
		w := int32(math.Max(1, math.Round(100*ctr/sumCTR)))
		weights[id] = w
	}
	return weights
}

func (s *Service) queryMABCreativeStats(ctx context.Context, from, to time.Time) (map[uuid.UUID]mabCreativeStat, error) {
	out := make(map[uuid.UUID]mabCreativeStat)

	impQuery := `
SELECT
    toString(campaign_id) AS campaign_id,
    nullIf(JSONExtractString(payload, 'creative_id'), '') AS creative_id,
    count() AS impressions
FROM impressions
WHERE created_at >= ? AND created_at < ?
GROUP BY campaign_id, creative_id`

	impRows, err := s.chQuery.Query(ctx, impQuery, from, to)
	if err != nil {
		return nil, fmt.Errorf("mab impressions query: %w", err)
	}
	defer impRows.Close()

	type key struct {
		campaignID string
		creativeID string
	}
	imps := make(map[key]int64)
	for impRows.Next() {
		var campaignID, creativeID string
		var impressions uint64
		if err := impRows.Scan(&campaignID, &creativeID, &impressions); err != nil {
			return nil, err
		}
		imps[key{campaignID: campaignID, creativeID: creativeID}] = int64(impressions)
	}
	if err := impRows.Err(); err != nil {
		return nil, err
	}

	clickQuery := `
SELECT
    toString(campaign_id) AS campaign_id,
    nullIf(JSONExtractString(payload, 'creative_id'), '') AS creative_id,
    count() AS clicks
FROM clicks
WHERE created_at >= ? AND created_at < ?
GROUP BY campaign_id, creative_id`

	clickRows, err := s.chQuery.Query(ctx, clickQuery, from, to)
	if err != nil {
		return nil, fmt.Errorf("mab clicks query: %w", err)
	}
	defer clickRows.Close()

	for clickRows.Next() {
		var campaignID, creativeID string
		var clicks uint64
		if err := clickRows.Scan(&campaignID, &creativeID, &clicks); err != nil {
			return nil, err
		}
		k := key{campaignID: campaignID, creativeID: creativeID}
		statKey, err := mabStatKey(campaignID, creativeID)
		if err != nil {
			continue
		}
		stat := out[statKey]
		stat.clicks = int64(clicks)
		stat.impressions = imps[k]
		out[statKey] = stat
	}
	if err := clickRows.Err(); err != nil {
		return nil, err
	}

	for k, impCount := range imps {
		statKey, err := mabStatKey(k.campaignID, k.creativeID)
		if err != nil {
			continue
		}
		if _, ok := out[statKey]; !ok {
			out[statKey] = mabCreativeStat{impressions: impCount}
		}
	}
	return out, nil
}

func mabStatKey(campaignID, creativeID string) (uuid.UUID, error) {
	if creativeID != "" {
		return uuid.Parse(creativeID)
	}
	return uuid.Parse(campaignID)
}

func (s *Service) AutoscaleBudgets(ctx context.Context, syncWorkers []*domain.SyncWorker) error {
	if s.cfg == nil {
		return nil
	}

	return s.withPgLow(ctx, func(runCtx context.Context) error {
		opCtx, cancel := workerContext(runCtx, workerBatchTimeout)
		defer cancel()

		for _, sw := range syncWorkers {
			if sw != nil {
				sw.SyncAll(opCtx)
			}
		}

		return pgx.BeginFunc(opCtx, s.GetPool(), func(tx pgx.Tx) error {
			return s.autoscaleBudgetsTx(opCtx, tx, nil)
		})
	})
}

func (s *Service) autoscaleBudgetsTx(ctx context.Context, tx pgx.Tx, merge deliveryOutboxMerge) error {
	if s.cfg == nil {
		return nil
	}

	q := db.New(tx)
	rows, err := q.GetAllActiveCampaignsWithStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch active campaigns with stats: %w", err)
	}

	byCustomer := make(map[uuid.UUID][]db.GetAllActiveCampaignsWithStatsRow)
	for _, row := range rows {
		custID := uuid.UUID(row.CustomerID.Bytes)
		byCustomer[custID] = append(byCustomer[custID], row)
	}

	for custID, campaigns := range byCustomer {
		if len(campaigns) < 2 {
			continue
		}

		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))`, custID.String()); err != nil {
			return fmt.Errorf("autoscale advisory lock for customer %s: %w", custID, err)
		}

		var bestCamp *db.GetAllActiveCampaignsWithStatsRow
		var bestCTR float64 = -1.0

		var worstCamp *db.GetAllActiveCampaignsWithStatsRow
		var worstCTR float64 = 2.0

		for i := range campaigns {
			c := &campaigns[i]
			if c.TotalImpressions <= 0 {
				continue
			}
			ctr := float64(c.TotalClicks) / float64(c.TotalImpressions)

			if ctr > s.cfg.AutoscaleHighCTRThreshold && c.TotalImpressions > s.cfg.AutoscaleMinImpressions {
				if ctr > bestCTR {
					bestCTR = ctr
					bestCamp = c
				}
			}

			limit := c.BudgetLimit
			spend := c.CurrentSpend
			remaining := limit - spend

			if ctr < s.cfg.AutoscaleLowCTRThreshold && remaining >= s.cfg.AutoscaleMinRemainingBudget {
				if ctr < worstCTR {
					worstCTR = ctr
					worstCamp = c
				}
			}
		}

		if bestCamp == nil || worstCamp == nil {
			continue
		}

		bestID := uuid.UUID(bestCamp.ID.Bytes)
		worstID := uuid.UUID(worstCamp.ID.Bytes)
		if bestID == worstID {
			continue
		}

		transferKey := autoscaleTransferKey(worstID, bestID, worstCamp, bestCamp)
		_, err := q.GetLedgerByHash(ctx, pgtype.Text{String: transferKey, Valid: true})
		if err == nil {
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("autoscale transfer idempotency check: %w", err)
		}

		var worstLocked, bestLocked db.Campaign

		if worstID.String() < bestID.String() {
			worstLocked, err = q.GetCampaignForUpdate(ctx, worstCamp.ID)
			if err != nil {
				return fmt.Errorf("failed to lock worst campaign %s: %w", worstID, err)
			}
			bestLocked, err = q.GetCampaignForUpdate(ctx, bestCamp.ID)
			if err != nil {
				return fmt.Errorf("failed to lock best campaign %s: %w", bestID, err)
			}
		} else {
			bestLocked, err = q.GetCampaignForUpdate(ctx, bestCamp.ID)
			if err != nil {
				return fmt.Errorf("failed to lock best campaign %s: %w", bestID, err)
			}
			worstLocked, err = q.GetCampaignForUpdate(ctx, worstCamp.ID)
			if err != nil {
				return fmt.Errorf("failed to lock worst campaign %s: %w", worstID, err)
			}
		}
		if worstLocked.Status != db.CampaignStatusTypeACTIVE || bestLocked.Status != db.CampaignStatusTypeACTIVE {
			continue
		}

		shiftAmount := s.cfg.AutoscaleShiftAmount
		worstLimit := worstLocked.BudgetLimit
		bestLimit := bestLocked.BudgetLimit

		newWorstLimit := worstLimit - shiftAmount
		newBestLimit := bestLimit + shiftAmount

		if newWorstLimit < worstLocked.CurrentSpend {
			continue
		}
		if newWorstLimit <= 0 {
			continue
		}

		_, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      domain.ToUUID(custID),
			Balance: shiftAmount,
		})
		if err != nil {
			return fmt.Errorf("failed to credit customer balance for autoscale release: %w", err)
		}

		_, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      domain.ToUUID(custID),
			CampaignID:      worstLocked.ID,
			Amount:          shiftAmount,
			Type:            db.LedgerTypeRELEASE,
			IdempotencyHash: pgtype.Text{String: transferKey + ":release", Valid: true},
			PaymentIntentID: pgtype.UUID{},
		})
		if err != nil {
			return fmt.Errorf("failed to record autoscale release ledger for campaign %s: %w", worstID, err)
		}

		_, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      domain.ToUUID(custID),
			Balance: -shiftAmount,
		})
		if err != nil {
			return fmt.Errorf("failed to debit customer balance for autoscale freeze: %w", err)
		}

		_, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      domain.ToUUID(custID),
			CampaignID:      bestLocked.ID,
			Amount:          shiftAmount,
			Type:            db.LedgerTypeFREEZE,
			IdempotencyHash: pgtype.Text{String: transferKey, Valid: true},
			PaymentIntentID: pgtype.UUID{},
		})
		if err != nil {
			return fmt.Errorf("failed to record autoscale freeze ledger for campaign %s: %w", bestID, err)
		}

		_, err = q.UpdateCampaignBudget(ctx, db.UpdateCampaignBudgetParams{
			ID:          worstLocked.ID,
			BudgetLimit: newWorstLimit,
		})
		if err != nil {
			return fmt.Errorf("failed to decrease budget for campaign %s: %w", worstID, err)
		}

		_, err = q.UpdateCampaignBudget(ctx, db.UpdateCampaignBudgetParams{
			ID:          bestLocked.ID,
			BudgetLimit: newBestLimit,
		})
		if err != nil {
			return fmt.Errorf("failed to increase budget for campaign %s: %w", bestID, err)
		}

		worstLimitStr := fmt.Sprintf("%.2f", float64(worstLimit)/1_000_000.0)
		newWorstLimitStr := fmt.Sprintf("%.2f", float64(newWorstLimit)/1_000_000.0)
		bestLimitStr := fmt.Sprintf("%.2f", float64(bestLimit)/1_000_000.0)
		newBestLimitStr := fmt.Sprintf("%.2f", float64(newBestLimit)/1_000_000.0)

		s.AuditLog(ctx, q, uuid.Nil, "AUTOSCALE_BUDGET_TRANSFER", "campaign", &worstID, map[string]any{
			"old_budget": worstLimitStr,
			"new_budget": newWorstLimitStr,
			"ctr":        worstCTR,
			"target":     bestID.String(),
		}, nil)

		s.AuditLog(ctx, q, uuid.Nil, "AUTOSCALE_BUDGET_TRANSFER", "campaign", &bestID, map[string]any{
			"old_budget": bestLimitStr,
			"new_budget": newBestLimitStr,
			"ctr":        bestCTR,
			"source":     worstID.String(),
		}, nil)

		worstPayload, err := coldpath.MarshalJSON(CampaignPayload{
			CampaignID:  worstID.String(),
			BudgetLimit: newWorstLimit,
		})
		if err != nil {
			return fmt.Errorf("marshal autoscale worst campaign outbox payload: %w", err)
		}
		bestPayload, err := coldpath.MarshalJSON(CampaignPayload{
			CampaignID:  bestID.String(),
			BudgetLimit: newBestLimit,
		})
		if err != nil {
			return fmt.Errorf("marshal autoscale best campaign outbox payload: %w", err)
		}

		if merge != nil {
			merge.upsert(worstID, outboxPriCreateCampaign, "CREATE_CAMPAIGN", worstPayload)
			merge.upsert(bestID, outboxPriCreateCampaign, "CREATE_CAMPAIGN", bestPayload)
		} else {
			_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
				EventType: "CREATE_CAMPAIGN",
				Payload:   worstPayload,
			})
			if err != nil {
				return fmt.Errorf("failed to create outbox event for worst campaign: %w", err)
			}

			_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
				EventType: "CREATE_CAMPAIGN",
				Payload:   bestPayload,
			})
			if err != nil {
				return fmt.Errorf("failed to create outbox event for best campaign: %w", err)
			}
		}
	}

	return nil
}

func autoscaleTransferKey(
	worstID, bestID uuid.UUID,
	worstCamp, bestCamp *db.GetAllActiveCampaignsWithStatsRow,
) string {
	return fmt.Sprintf(
		"autoscale-transfer:%s:%s:%d:%d:%d:%d",
		worstID, bestID,
		worstCamp.TotalImpressions, worstCamp.TotalClicks,
		bestCamp.TotalImpressions, bestCamp.TotalClicks,
	)
}

type BlacklistDTO struct {
	ID        int64  `json:"id"`
	IP        string `json:"ip"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

func (s *Service) BlockIP(ctx context.Context, ip string, source string) error {
	return s.BlockIPWithTTL(ctx, ip, source, nil)
}

func (s *Service) PreviewBlockIP(ctx context.Context, ip string, source string, ttlSeconds *int64) (MutationPreview, error) {
	return s.blockIPWithTTL(ctx, ip, source, ttlSeconds, true)
}

func (s *Service) BlockIPWithTTL(ctx context.Context, ip string, source string, ttlSeconds *int64) error {
	_, err := s.blockIPWithTTL(ctx, ip, source, ttlSeconds, false)
	return err
}

func (s *Service) blockIPWithTTL(ctx context.Context, ip string, source string, ttlSeconds *int64, dryRun bool) (MutationPreview, error) {
	if allowlist.IsProtected(ip) {
		return MutationPreview{}, fmt.Errorf("IP %s is protected by allowlist", ip)
	}

	reason := normalizeBlacklistReason(source)
	expiresAt := resolveBlacklistExpiry(reason, ttlSeconds, blacklistTTLFromConfig(s.cfg))

	if dryRun {
		preview := MutationPreview{
			DryRun: true,
			Action: "BLOCK_IP",
			WouldChange: map[string]any{
				"ip":           ip,
				"reason":       reason,
				"outbox_event": "UPDATE_BLACKLIST",
				"action":       "add",
			},
		}
		if expiresAt.Valid {
			preview.WouldChange["expires_at"] = expiresAt.Time.UTC().Format(time.RFC3339)
		}
		return preview, nil
	}

	err := pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		_, err := q.CreateBlacklistIP(ctx, db.CreateBlacklistIPParams{
			Ip:        ip,
			Reason:    reason,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			return err
		}

		var ttlVal pgtype.Int4
		if expiresAt.Valid {
			diff := expiresAt.Time.Sub(time.Now().UTC())
			if diff > 0 {
				ttlVal = pgtype.Int4{Int32: int32(diff.Seconds()), Valid: true}
			}
		}

		_, err = q.CreateEdgeBlockAudit(ctx, db.CreateEdgeBlockAuditParams{
			Ip:       ip,
			ReasonID: reason,
			Ttl:      ttlVal,
			Source:   source,
		})
		if err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "BLOCK_IP", "system", nil, map[string]string{"ip": ip, "source": reason}, nil)

		payload, err := coldpath.MarshalJSON(BlacklistPayload{Action: "add", IP: ip, Reason: reason})
		if err != nil {
			return fmt.Errorf("marshal blacklist outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "UPDATE_BLACKLIST",
			Payload:   payload,
		})
		return err
	})
	return MutationPreview{}, err
}

func (s *Service) EnqueueFraudThreat(ctx context.Context, p FraudThreatPayload) error {
	if _, err := uuid.Parse(p.CampaignID); err != nil {
		return fmt.Errorf("invalid campaign id: %w", err)
	}

	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		payload, err := coldpath.MarshalJSON(p)
		if err != nil {
			return fmt.Errorf("marshal ml threat payload: %w", err)
		}

		var eventType string
		switch p.Action {
		case "boost":
			eventType = "ML_SCORE_BOOST"
		case "ghost":
			eventType = "ML_GHOST_IVT"
		case "blacklist":
			eventType = "ML_BLACKLIST_ADD"
		default:
			return fmt.Errorf("unknown ml threat action: %s", p.Action)
		}

		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: eventType,
			Payload:   payload,
		})
		return err
	})
}

func (s *Service) UnblockIP(ctx context.Context, ip string, source string) error {
	reason := normalizeBlacklistReason(source)

	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		err := q.DeleteBlacklistIP(ctx, ip)
		if err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "UNBLOCK_IP", "system", nil, map[string]string{"ip": ip, "source": reason}, nil)

		payload, err := coldpath.MarshalJSON(BlacklistPayload{Action: "remove", IP: ip, Reason: reason})
		if err != nil {
			return fmt.Errorf("marshal blacklist outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "UPDATE_BLACKLIST",
			Payload:   payload,
		})
		return err
	})
}

func (s *Service) UpdateSettings(ctx context.Context, settings map[string]string) error {
	normalized, err := normalizeSystemSettings(settings)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		for k, v := range normalized {
			err := q.SetSystemSetting(ctx, db.SetSystemSettingParams{
				Key:   k,
				Value: v,
			})
			if err != nil {
				return err
			}
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "UPDATE_SETTINGS", "system", nil, normalized, nil)
		payloadBytes, err := coldpath.MarshalJSON(SettingsPayload{Settings: normalized})
		if err != nil {
			return fmt.Errorf("marshal settings outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "UPDATE_SETTINGS", Payload: payloadBytes})
		return err
	})
}

func normalizeSystemSettings(settings map[string]string) (map[string]string, error) {
	if len(settings) == 0 {
		return settings, nil
	}
	out := make(map[string]string, len(settings))
	for k, v := range settings {
		if k == "rtb_budget_authority" {
			norm, err := domain.NormalizeRtbBudgetAuthoritySetting(v)
			if err != nil {
				return nil, err
			}
			out[k] = norm
			continue
		}
		if k == domain.SystemSettingRtbMode {
			norm, err := domain.NormalizeRtbModeSetting(v)
			if err != nil {
				return nil, err
			}
			out[k] = norm
			continue
		}
		out[k] = v
	}
	return out, nil
}

func (s *Service) ListBlacklist(ctx context.Context, limit, offset int32) ([]BlacklistDTO, int64, error) {
	q := db.New(s.GetPool())
	listParams := db.ListBlacklistParams{Limit: limit, Offset: offset}
	return coldpath.PaginatedList(
		func() (int64, error) { return q.CountBlacklist(ctx) },
		func() ([]db.IpBlacklist, error) { return q.ListBlacklist(ctx, listParams) },
		blacklistToDTO,
	)
}

func blacklistToDTO(r db.IpBlacklist) BlacklistDTO {
	dto := BlacklistDTO{
		ID:        r.ID,
		IP:        r.Ip,
		Reason:    r.Reason,
		CreatedAt: r.CreatedAt.Time.Format(time.RFC3339),
	}
	if r.ExpiresAt.Valid {
		dto.ExpiresAt = r.ExpiresAt.Time.UTC().Format(time.RFC3339)
	}
	return dto
}

func (s *Service) GetSettings(ctx context.Context) (map[string]string, error) {
	q := db.New(s.GetPool())
	rows, err := q.GetAllSystemSettings(ctx)
	if err != nil {
		return nil, err
	}
	return coldpath.KeyByValue(rows, func(r db.GetAllSystemSettingsRow) string { return r.Key }, func(r db.GetAllSystemSettingsRow) string { return r.Value }), nil
}

func (s *Service) SyncSystemState(ctx context.Context) error {
	q := db.New(s.GetPool())

	bl, err := q.GetAllBlacklist(ctx)
	if err != nil {
		return fmt.Errorf("failed to get blacklist from db: %w", err)
	}

	if len(s.rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}

	reasonIPs := make(map[string][]any)
	for _, item := range bl {
		reason := normalizeBlacklistReason(item.Reason)
		reasonIPs[reason] = append(reasonIPs[reason], item.Ip)
	}

	for reason, ips := range reasonIPs {
		key := "blacklist:" + reason
		for _, rdb := range s.rdbs {
			if err := rdb.Del(ctx, key).Err(); err != nil {
				return fmt.Errorf("failed to reset blacklist key %s: %w", key, err)
			}
			if len(ips) > 0 {
				if err := rdb.SAdd(ctx, key, ips...).Err(); err != nil {
					return fmt.Errorf("failed to sync blacklist key %s: %w", key, err)
				}
			}
		}
	}

	st, err := q.GetAllSystemSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get settings from db: %w", err)
	}

	if len(st) > 0 {
		settingsMap := make(map[string]string)
		for _, r := range st {
			settingsMap[r.Key] = r.Value
		}
		if err := syncGlobalConfigToAllShards(ctx, s.rdbs, settingsMap, 0); err != nil {
			return fmt.Errorf("failed to sync settings to redis: %w", err)
		}
		if err := replicateConfigVersionFromPrimary(ctx, s.rdbs); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) ToggleEmergencyBreaker(ctx context.Context, active bool, reason string) error {
	val := "false"
	if active {
		val = "true"
	}

	err := pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		err := q.SetSystemSetting(ctx, db.SetSystemSettingParams{
			Key:   "emergency_breaker",
			Value: val,
		})
		if err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}

		s.AuditLog(ctx, q, uid, "EMERGENCY_BREAKER_TOGGLED", "system", nil, map[string]any{
			"active": active,
			"reason": reason,
		}, nil)

		settings := map[string]string{
			"emergency_breaker": val,
		}
		payloadBytes, err := coldpath.MarshalJSON(SettingsPayload{Settings: settings})
		if err != nil {
			return fmt.Errorf("marshal emergency breaker outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "UPDATE_SETTINGS",
			Payload:   payloadBytes,
		})
		return err
	})
	return err
}
