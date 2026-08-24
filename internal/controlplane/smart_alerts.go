package controlplane

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/branding"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	smartAlertCHTimeout       = 15 * time.Second
	smartAlertWebhookTimeout  = 45 * time.Second
	smartAlertMaxWebhookBytes = 64 << 10
)

var validAlertMetrics = map[string]struct{}{
	"clicks":     {},
	"cr":         {},
	"roi_pct":    {},
	"bot_clicks": {},
}

var validAlertOperators = map[string]struct{}{
	"gt":  {},
	"lt":  {},
	"gte": {},
	"lte": {},
}

func normalizeAlertMetric(metric string) (string, error) {
	m := strings.ToLower(strings.TrimSpace(metric))
	if _, ok := validAlertMetrics[m]; !ok {
		return "", fmt.Errorf("invalid metric %q", metric)
	}
	return m, nil
}

func normalizeAlertOperator(op string) (string, error) {
	o := strings.ToLower(strings.TrimSpace(op))
	if _, ok := validAlertOperators[o]; !ok {
		return "", fmt.Errorf("invalid operator %q", op)
	}
	return o, nil
}

func clampAlertWindowMinutes(window int) int {
	if window < 5 {
		return 5
	}
	if window > 1440 {
		return 1440
	}
	return window
}

func alertWindowBounds(now time.Time, windowMinutes int) (start, end time.Time) {
	end = now.UTC().Truncate(time.Minute)
	start = end.Add(-time.Duration(windowMinutes) * time.Minute)
	return start, end
}

func alertThresholdBreached(operator string, observed, threshold float64) bool {
	switch operator {
	case "gt":
		return observed > threshold
	case "gte":
		return observed >= threshold
	case "lt":
		return observed < threshold
	case "lte":
		return observed <= threshold
	default:
		return false
	}
}

func (s *Service) ListSmartAlertRules(ctx context.Context, customerID uuid.UUID) ([]SmartAlertRuleDTO, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, customer_id, campaign_id, name, metric, operator, threshold,
		 window_minutes, webhook_url, enabled, created_at, updated_at
		FROM alert_rules
		WHERE customer_id = $1
		ORDER BY created_at DESC`, domain.ToUUID(customerID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SmartAlertRuleDTO
	for rows.Next() {
		dto, err := scanSmartAlertRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, rows.Err()
}

func (s *Service) CreateSmartAlertRule(ctx context.Context, req UpsertSmartAlertRuleRequest) (SmartAlertRuleDTO, error) {
	if s == nil || s.pool == nil {
		return SmartAlertRuleDTO{}, fmt.Errorf("service unavailable")
	}
	customerID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		return SmartAlertRuleDTO{}, fmt.Errorf("invalid customer_id")
	}
	metric, err := normalizeAlertMetric(req.Metric)
	if err != nil {
		return SmartAlertRuleDTO{}, err
	}
	operator, err := normalizeAlertOperator(req.Operator)
	if err != nil {
		return SmartAlertRuleDTO{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return SmartAlertRuleDTO{}, fmt.Errorf("name is required")
	}
	webhookURL := strings.TrimSpace(req.WebhookURL)
	if webhookURL == "" || !strings.HasPrefix(webhookURL, "http") {
		return SmartAlertRuleDTO{}, fmt.Errorf("webhook_url must be an http(s) URL")
	}
	window := clampAlertWindowMinutes(req.WindowMinutes)
	if req.WindowMinutes == 0 {
		window = 60
	}

	var campParam pgtype.UUID
	if strings.TrimSpace(req.CampaignID) != "" {
		campID, err := uuid.Parse(req.CampaignID)
		if err != nil {
			return SmartAlertRuleDTO{}, fmt.Errorf("invalid campaign_id")
		}
		campParam = domain.ToUUID(campID)
	}

	row := s.pool.QueryRow(ctx, `
		INSERT INTO alert_rules (
			customer_id, campaign_id, name, metric, operator, threshold,
			window_minutes, webhook_url, enabled
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, customer_id, campaign_id, name, metric, operator, threshold,
		 window_minutes, webhook_url, enabled, created_at, updated_at`,
		domain.ToUUID(customerID), campParam, name, metric, operator, req.Threshold,
		window, webhookURL, req.Enabled,
	)
	return scanSmartAlertRule(row)
}

func (s *Service) UpdateSmartAlertRule(ctx context.Context, ruleID uuid.UUID, req UpsertSmartAlertRuleRequest) (SmartAlertRuleDTO, error) {
	if s == nil || s.pool == nil {
		return SmartAlertRuleDTO{}, fmt.Errorf("service unavailable")
	}
	metric, err := normalizeAlertMetric(req.Metric)
	if err != nil {
		return SmartAlertRuleDTO{}, err
	}
	operator, err := normalizeAlertOperator(req.Operator)
	if err != nil {
		return SmartAlertRuleDTO{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return SmartAlertRuleDTO{}, fmt.Errorf("name is required")
	}
	webhookURL := strings.TrimSpace(req.WebhookURL)
	if webhookURL == "" || !strings.HasPrefix(webhookURL, "http") {
		return SmartAlertRuleDTO{}, fmt.Errorf("webhook_url must be an http(s) URL")
	}
	window := clampAlertWindowMinutes(req.WindowMinutes)
	if req.WindowMinutes == 0 {
		window = 60
	}

	var campParam pgtype.UUID
	if strings.TrimSpace(req.CampaignID) != "" {
		campID, err := uuid.Parse(req.CampaignID)
		if err != nil {
			return SmartAlertRuleDTO{}, fmt.Errorf("invalid campaign_id")
		}
		campParam = domain.ToUUID(campID)
	}

	row := s.pool.QueryRow(ctx, `
		UPDATE alert_rules
		SET name = $2, campaign_id = $3, metric = $4, operator = $5, threshold = $6,
		 window_minutes = $7, webhook_url = $8, enabled = $9, updated_at = now()
		WHERE id = $1
		RETURNING id, customer_id, campaign_id, name, metric, operator, threshold,
		 window_minutes, webhook_url, enabled, created_at, updated_at`,
		domain.ToUUID(ruleID), name, campParam, metric, operator, req.Threshold,
		window, webhookURL, req.Enabled,
	)
	dto, err := scanSmartAlertRule(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SmartAlertRuleDTO{}, fmt.Errorf("rule not found")
		}
		return SmartAlertRuleDTO{}, err
	}
	return dto, nil
}

func (s *Service) DeleteSmartAlertRule(ctx context.Context, ruleID uuid.UUID) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("service unavailable")
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM alert_rules WHERE id = $1`, domain.ToUUID(ruleID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}

func (s *Service) ListSmartAlertHistory(ctx context.Context, customerID uuid.UUID, limit int) ([]SmartAlertEventDTO, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, rule_id, customer_id, campaign_id, window_start, window_end,
		 metric, operator, threshold, observed_value, webhook_status, webhook_error,
		 fired_at, acked_at, acked_by
		FROM alert_rule_events
		WHERE customer_id = $1
		ORDER BY fired_at DESC
		LIMIT $2`, domain.ToUUID(customerID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SmartAlertEventDTO
	for rows.Next() {
		dto, err := scanSmartAlertEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, rows.Err()
}

func (s *Service) AckSmartAlertEvent(ctx context.Context, eventID, actorID uuid.UUID) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("service unavailable")
	}
	var actor pgtype.UUID
	if actorID != uuid.Nil {
		actor = domain.ToUUID(actorID)
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE alert_rule_events
		SET acked_at = now(), acked_by = COALESCE($2, acked_by)
		WHERE id = $1 AND acked_at IS NULL`,
		domain.ToUUID(eventID), actor)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("event not found or already acked")
	}
	return nil
}

type smartAlertRuleRow struct {
	ID            uuid.UUID
	CustomerID    uuid.UUID
	CampaignID    uuid.UUID
	HasCampaign   bool
	Name          string
	Metric        string
	Operator      string
	Threshold     float64
	WindowMinutes int
	WebhookURL    string
}

type SmartAlertsWorker struct {
	svc      *Service
	interval time.Duration
	client   *http.Client
}

func NewSmartAlertsWorker(svc *Service, interval time.Duration) *SmartAlertsWorker {
	if interval < 5*time.Minute {
		interval = 5 * time.Minute
	}
	if interval > 60*time.Minute {
		interval = 60 * time.Minute
	}
	return &SmartAlertsWorker{
		svc:      svc,
		interval: interval,
		client:   &http.Client{Timeout: smartAlertWebhookTimeout},
	}
}

func (s *Service) StartSmartAlertsWorker(ctx context.Context, interval time.Duration) {
	if s == nil || s.cfg == nil || !s.cfg.Management.SmartAlertsEnabled {
		return
	}
	w := NewSmartAlertsWorker(s, interval)
	s.StartBackgroundWorker(func() {
		w.Start(ctx)
	})
	slog.Info("smart alerts worker enabled", "interval", w.interval)
}

func (w *SmartAlertsWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *SmartAlertsWorker) tick(ctx context.Context) {
	if w == nil || w.svc == nil || w.svc.pool == nil {
		return
	}
	ch := w.svc.CHQuery()
	if ch == nil {
		slog.Debug("smart alerts: clickhouse not configured, skip tick")
		return
	}
	rules, err := w.loadEnabledRules(ctx)
	if err != nil {
		slog.Error("smart alerts: load rules", "err", err)
		return
	}
	now := time.Now().UTC()
	if err := w.evaluateRulesBatch(ctx, ch, rules, now); err != nil {
		slog.Error("smart alerts: evaluate rules", "err", err)
	}
}

func (w *SmartAlertsWorker) loadEnabledRules(ctx context.Context) ([]smartAlertRuleRow, error) {
	rows, err := w.svc.pool.Query(ctx, `
		SELECT id, customer_id, campaign_id, name, metric, operator, threshold,
		 window_minutes, webhook_url
		FROM alert_rules
		WHERE enabled = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []smartAlertRuleRow
	for rows.Next() {
		var r smartAlertRuleRow
		var campID pgtype.UUID
		if err := rows.Scan(
			&r.ID, &r.CustomerID, &campID, &r.Name, &r.Metric, &r.Operator,
			&r.Threshold, &r.WindowMinutes, &r.WebhookURL,
		); err != nil {
			return nil, err
		}
		if campID.Valid {
			r.CampaignID = uuid.UUID(campID.Bytes)
			r.HasCampaign = true
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (w *SmartAlertsWorker) deliverWebhook(ctx context.Context, url string, body []byte) (status, errMsg string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "failed", err.Error()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", branding.HTTPUserAgent("SmartAlerts"))

	resp, err := w.client.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return "failed", err.Error()
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, smartAlertMaxWebhookBytes))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "failed", fmt.Sprintf("http %d", resp.StatusCode)
	}
	return "delivered", ""
}

func roundAlertFloat(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*10000) / 10000
}

type smartAlertRowScanner interface {
	Scan(dest ...any) error
}

func scanSmartAlertRule(row smartAlertRowScanner) (SmartAlertRuleDTO, error) {
	var dto SmartAlertRuleDTO
	var id, customerID uuid.UUID
	var campID pgtype.UUID
	if err := row.Scan(
		&id, &customerID, &campID, &dto.Name, &dto.Metric, &dto.Operator,
		&dto.Threshold, &dto.WindowMinutes, &dto.WebhookURL, &dto.Enabled,
		&dto.CreatedAt, &dto.UpdatedAt,
	); err != nil {
		return SmartAlertRuleDTO{}, err
	}
	dto.ID = id.String()
	dto.CustomerID = customerID.String()
	dto.CampaignID = formatOptionalUUID(campID)
	return dto, nil
}

func scanSmartAlertEvent(row smartAlertRowScanner) (SmartAlertEventDTO, error) {
	var dto SmartAlertEventDTO
	var id, ruleID, customerID uuid.UUID
	var campID, ackedBy pgtype.UUID
	var ackedAt pgtype.Timestamptz
	if err := row.Scan(
		&id, &ruleID, &customerID, &campID,
		&dto.WindowStart, &dto.WindowEnd, &dto.Metric, &dto.Operator,
		&dto.Threshold, &dto.ObservedValue, &dto.WebhookStatus, &dto.WebhookError,
		&dto.FiredAt, &ackedAt, &ackedBy,
	); err != nil {
		return SmartAlertEventDTO{}, err
	}
	dto.ID = id.String()
	dto.RuleID = ruleID.String()
	dto.CustomerID = customerID.String()
	dto.CampaignID = formatOptionalUUID(campID)
	if ackedAt.Valid {
		t := ackedAt.Time
		dto.AckedAt = &t
	}
	dto.AckedBy = formatOptionalUUID(ackedBy)
	return dto, nil
}
