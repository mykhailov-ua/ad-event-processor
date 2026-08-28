package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/platformsync"
	"ad-event-processor/pkg/branding"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ExecutorHost interface {
	Pool() *pgxpool.Pool
	PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error
	BlacklistPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error
}

type serviceExecutor struct {
	host   ExecutorHost
	client *http.Client
}

func NewExecutor(host ExecutorHost) Executor {
	return &serviceExecutor{
		host:   host,
		client: &http.Client{Timeout: webhookTimeout},
	}
}

func (e *serviceExecutor) Notify(ctx context.Context, webhookURL string, payload []byte) (string, string) {
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

func (e *serviceExecutor) PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	if e == nil || e.host == nil {
		return fmt.Errorf("service unavailable")
	}
	return e.host.PauseCampaign(ctx, campaignID, reason)
}

func (e *serviceExecutor) BlacklistPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	if e == nil || e.host == nil {
		return fmt.Errorf("service unavailable")
	}
	return e.host.BlacklistPlacement(ctx, campaignID, placementID)
}

func (e *serviceExecutor) PlatformPause(ctx context.Context, _ uuid.UUID, campaignID uuid.UUID, network, idempotencyKey string) error {
	if e == nil || e.host == nil {
		return fmt.Errorf("service unavailable")
	}
	pool := e.host.Pool()
	if pool == nil {
		return fmt.Errorf("service unavailable")
	}
	network = platformsync.NormalizeNetwork(network)
	if !platformsync.NetworkSupported(network) {
		return fmt.Errorf("unsupported network %q", network)
	}
	q := db.New(pool)
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
