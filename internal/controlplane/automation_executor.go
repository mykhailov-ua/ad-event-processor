package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"ad-event-processor/internal/automation"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/platformsync"
	"ad-event-processor/pkg/branding"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
)

type automationExecutor struct {
	svc    *Service
	client *http.Client
}

func (s *Service) newAutomationExecutor() automation.Executor {
	return &automationExecutor{
		svc:    s,
		client: &http.Client{Timeout: 45 * time.Second},
	}
}

func (e *automationExecutor) Notify(ctx context.Context, webhookURL string, payload []byte) (string, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return "failed", err.Error()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", branding.HTTPUserAgent("AutomationRules"))
	resp, err := e.client.Do(req)
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

func (e *automationExecutor) PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	if e == nil || e.svc == nil {
		return fmt.Errorf("service unavailable")
	}
	return e.svc.PauseCampaign(ctx, campaignID, reason)
}

func (e *automationExecutor) BlacklistPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	if e == nil || e.svc == nil {
		return fmt.Errorf("service unavailable")
	}
	return e.svc.BlockCampaignPlacement(ctx, campaignID, placementID)
}

func (e *automationExecutor) PlatformPause(ctx context.Context, _ uuid.UUID, campaignID uuid.UUID, network, idempotencyKey string) error {
	if e == nil || e.svc == nil {
		return fmt.Errorf("service unavailable")
	}
	network = platformsync.NormalizeNetwork(network)
	if !platformsync.NetworkSupported(network) {
		return fmt.Errorf("unsupported network %q", network)
	}
	q := db.New(e.svc.pool)
	link, err := q.GetPlatformCampaignLink(ctx, db.GetPlatformCampaignLinkParams{
		CampaignID: domain.ToUUID(campaignID),
		Network:    network,
	})
	if err != nil {
		return fmt.Errorf("platform link not found: %w", err)
	}
	reqJSON, err := json.Marshal(platformsync.MutationRequest{})
	if err != nil {
		return err
	}
	_, err = q.InsertPlatformCampaignMutation(ctx, db.InsertPlatformCampaignMutationParams{
		IdempotencyKey: idempotencyKey,
		CampaignID:     domain.ToUUID(campaignID),
		CustomerID:     link.CustomerID,
		Network:        network,
		Action:         platformsync.ActionPause,
		RequestJson:    reqJSON,
		Status:         platformsync.MutationPending,
	})
	return err
}

func (s *Service) StartAutomationWorker(ctx context.Context, intervalMinutes int) {
	if s == nil || s.cfg == nil || !s.cfg.Management.AutomationRulesEnabled {
		return
	}
	clickhouseQuery := s.ClickHouseQuery()
	if clickhouseQuery == nil {
		return
	}
	interval := time.Duration(intervalMinutes) * time.Minute
	maxEvals := 50
	if s.cfg.Management.AutomationRulesMaxEvalsPerCustomerPerTick > 0 {
		maxEvals = s.cfg.Management.AutomationRulesMaxEvalsPerCustomerPerTick
	}
	w := automation.NewWorker(s.pool, clickhouseQuery, s.newAutomationExecutor(), interval, maxEvals)
	s.StartBackgroundWorker(func() {
		w.Start(ctx)
	})
	slog.Info("automation rules worker enabled", "interval", interval)
}
