package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/branding"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const webhookTimeout = 45 * time.Second

type Executor interface {
	Notify(ctx context.Context, webhookURL string, payload []byte) (status, errMsg string)
	PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error
	BlacklistPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error
	PlatformPause(ctx context.Context, customerID, campaignID uuid.UUID, network, idempotencyKey string) error
}

type Worker struct {
	pool     *pgxpool.Pool
	ch       *database.CHQuery
	exec     Executor
	interval time.Duration
	client   *http.Client
}

func NewWorker(pool *pgxpool.Pool, ch *database.CHQuery, exec Executor, interval time.Duration) *Worker {
	if interval < 5*time.Minute {
		interval = 15 * time.Minute
	}
	if interval > 60*time.Minute {
		interval = 60 * time.Minute
	}
	return &Worker{
		pool:     pool,
		ch:       ch,
		exec:     exec,
		interval: interval,
		client:   &http.Client{Timeout: webhookTimeout},
	}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	w.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	if w == nil || w.pool == nil || w.ch == nil {
		return
	}
	rules, err := db.New(w.pool).ListEnabledAutomationRules(ctx)
	if err != nil {
		slog.Error("automation: list rules", "error", err)
		return
	}
	if len(rules) == 0 {
		return
	}
	now := time.Now().UTC()
	campaignsByCustomer, err := w.campaignsByCustomer(ctx, rules)
	if err != nil {
		slog.Error("automation: list campaigns", "error", err)
		return
	}
	for _, row := range rules {
		rule, err := ruleFromRow(row)
		if err != nil {
			slog.Warn("automation: skip rule", "rule_id", row.ID, "error", err)
			continue
		}
		if rule.HasLastFired && now.Sub(rule.LastFiredAt) < time.Duration(rule.CooldownMinutes)*time.Minute {
			continue
		}
		campaignIDs, err := resolveCampaignIDs(rule, campaignsByCustomer)
		if err != nil || len(campaignIDs) == 0 {
			continue
		}
		matches, err := EvaluateRule(ctx, w.ch, rule, campaignIDs, now)
		if err != nil {
			slog.Warn("automation: evaluate", "rule_id", rule.ID, "error", err)
			continue
		}
		for _, match := range matches {
			if err := w.applyMatch(ctx, rule, match); err != nil {
				slog.Warn("automation: apply match", "rule_id", rule.ID, "error", err)
			}
		}
	}
}

func (w *Worker) DryRun(ctx context.Context, rule Rule, campaignIDs []uuid.UUID) ([]WouldFire, error) {
	if w == nil || w.ch == nil {
		return nil, fmt.Errorf("clickhouse not configured")
	}
	matches, err := EvaluateRule(ctx, w.ch, rule, campaignIDs, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return MatchesToWouldFire(matches), nil
}

func (w *Worker) applyMatch(ctx context.Context, rule Rule, match Match) error {
	payload, _ := json.Marshal(map[string]any{
		"rule_id":        match.RuleID.String(),
		"customer_id":    match.CustomerID.String(),
		"campaign_id":    match.CampaignID.String(),
		"placement_id":   match.PlacementID,
		"metric":         match.Metric,
		"operator":       match.Operator,
		"threshold":      match.Threshold,
		"observed_value": match.ObservedValue,
		"window_start":   match.WindowStart.Format(time.RFC3339),
		"window_end":     match.WindowEnd.Format(time.RFC3339),
	})

	for _, action := range match.Actions {
		hash := ActionHash(match.RuleID, match.CampaignID, match.PlacementID, match.WindowEnd, action.Type)
		tag, err := db.New(w.pool).InsertAutomationRuleFire(ctx, db.InsertAutomationRuleFireParams{
			RuleID:        pgtype.UUID{Bytes: match.RuleID, Valid: true},
			ActionHash:    hash,
			CampaignID:    pgtype.UUID{Bytes: match.CampaignID, Valid: true},
			PlacementID:   match.PlacementID,
			Metric:        match.Metric,
			ObservedValue: match.ObservedValue,
			Payload:       payload,
		})
		if err != nil {
			return err
		}
		if tag == 0 {
			continue
		}
		if err := w.runAction(ctx, match, action, payload, hash); err != nil {
			return err
		}
	}
	_ = db.New(w.pool).UpdateAutomationRuleLastFired(ctx, db.UpdateAutomationRuleLastFiredParams{
		ID:          pgtype.UUID{Bytes: rule.ID, Valid: true},
		LastFiredAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
	return nil
}

func (w *Worker) runAction(ctx context.Context, match Match, action Action, payload []byte, idempotencyKey string) error {
	if w.exec == nil {
		return nil
	}
	switch action.Type {
	case ActionNotify:
		url := action.WebhookURL
		if url == "" {
			return nil
		}
		status, errMsg := w.notify(ctx, url, payload)
		if status != "delivered" {
			return fmt.Errorf("notify %s: %s", status, errMsg)
		}
	case ActionPauseCampaign:
		return w.exec.PauseCampaign(ctx, match.CampaignID, "automation_rule")
	case ActionBlacklistPlacement:
		if match.PlacementID == "" {
			return nil
		}
		return w.exec.BlacklistPlacement(ctx, match.CampaignID, match.PlacementID)
	case ActionPlatformPause:
		network := action.Network
		if network == "" {
			return nil
		}
		return w.exec.PlatformPause(ctx, match.CustomerID, match.CampaignID, network, idempotencyKey)
	default:
		return fmt.Errorf("unsupported action %q", action.Type)
	}
	return nil
}

func (w *Worker) notify(ctx context.Context, url string, body []byte) (status, errMsg string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "failed", err.Error()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", branding.HTTPUserAgent("AutomationRules"))
	resp, err := w.client.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return "failed", err.Error()
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "failed", fmt.Sprintf("http %d", resp.StatusCode)
	}
	return "delivered", ""
}

func (w *Worker) campaignsByCustomer(ctx context.Context, rules []db.AutomationRule) (map[uuid.UUID][]uuid.UUID, error) {
	need := make(map[uuid.UUID]struct{})
	for _, row := range rules {
		if !row.CampaignID.Valid {
			need[uuid.UUID(row.CustomerID.Bytes)] = struct{}{}
		}
	}
	out := make(map[uuid.UUID][]uuid.UUID)
	if len(need) == 0 {
		return out, nil
	}
	customerIDs := make([]pgtype.UUID, 0, len(need))
	for id := range need {
		customerIDs = append(customerIDs, pgtype.UUID{Bytes: id, Valid: true})
	}
	rows, err := db.New(w.pool).ListCampaignIDsByCustomers(ctx, customerIDs)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		customerID := uuid.UUID(row.CustomerID.Bytes)
		campaignID := uuid.UUID(row.CampaignID.Bytes)
		out[customerID] = append(out[customerID], campaignID)
	}
	return out, nil
}

func resolveCampaignIDs(rule Rule, campaignsByCustomer map[uuid.UUID][]uuid.UUID) ([]uuid.UUID, error) {
	if rule.HasCampaign {
		return []uuid.UUID{rule.CampaignID}, nil
	}
	return campaignsByCustomer[rule.CustomerID], nil
}

func ruleFromRow(row db.AutomationRule) (Rule, error) {
	actions, err := ParseActions(row.Actions)
	if err != nil {
		return Rule{}, err
	}
	metric, err := NormalizeMetric(row.Metric)
	if err != nil {
		return Rule{}, err
	}
	rule := Rule{
		ID:              uuid.UUID(row.ID.Bytes),
		CustomerID:      uuid.UUID(row.CustomerID.Bytes),
		Name:            row.Name,
		Metric:          metric,
		Operator:        row.Operator,
		Threshold:       row.Threshold,
		WindowMinutes:   int(row.WindowMinutes),
		GroupBy:         row.GroupBy,
		Actions:         actions,
		CooldownMinutes: int(row.CooldownMinutes),
		Enabled:         row.Enabled,
	}
	if row.CampaignID.Valid {
		rule.CampaignID = uuid.UUID(row.CampaignID.Bytes)
		rule.HasCampaign = true
	}
	if row.LastFiredAt.Valid {
		rule.LastFiredAt = row.LastFiredAt.Time
		rule.HasLastFired = true
	}
	return rule, nil
}

func RuleFromRow(row db.AutomationRule) (Rule, error) {
	return ruleFromRow(row)
}
