package platformsync

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"ad-event-processor/internal/costsync"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	platformAdvisoryLockKey = int64(0x6164705f706c6174)
	syncBatchSize           = 200
)

type Worker struct {
	pool          *pgxpool.Pool
	encryptionKey []byte
	httpClient    *http.Client
	costWorker    *costsync.Worker
	networkBase   map[string]string
	cycleWG       sync.WaitGroup
}

func NewWorker(pool *pgxpool.Pool, encryptionKey []byte, costWorker *costsync.Worker) *Worker {
	return &Worker{
		pool:          pool,
		encryptionKey: encryptionKey,
		httpClient:    &http.Client{Timeout: 90 * time.Second},
		costWorker:    costWorker,
		networkBase:   map[string]string{},
	}
}

func (w *Worker) Start(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	w.cycleWG.Add(1)
	go func() {
		defer w.cycleWG.Done()
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		w.runCycle(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.runCycle(ctx)
			}
		}
	}()
}

func (w *Worker) runCycle(ctx context.Context) {
	if w.pool == nil {
		return
	}
	conn, err := w.pool.Acquire(ctx)
	if err != nil {
		slog.Warn("platformsync: acquire conn", "error", err)
		return
	}
	defer conn.Release()

	var locked bool
	if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", platformAdvisoryLockKey).Scan(&locked); err != nil || !locked {
		return
	}
	defer func() {
		_, _ = conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", platformAdvisoryLockKey)
	}()

	w.processPendingMutations(ctx)
	w.syncLinkStatuses(ctx)
}

func (w *Worker) RunManual(ctx context.Context, campaignID uuid.UUID) error {
	if w == nil || w.pool == nil {
		return fmt.Errorf("platformsync: worker not configured")
	}
	q := db.New(w.pool)
	rows, err := q.ListPlatformCampaignLinks(ctx, db.ListPlatformCampaignLinksParams{
		Column1: pgtype.UUID{Bytes: campaignID, Valid: true},
	})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if syncErr := w.syncOneLink(ctx, q, row); syncErr != nil {
			return syncErr
		}
	}
	return nil
}

func (w *Worker) syncLinkStatuses(ctx context.Context) {
	q := db.New(w.pool)
	rows, err := q.ListPlatformCampaignLinksForSync(ctx, int32(syncBatchSize))
	if err != nil {
		slog.Warn("platformsync: list links", "error", err)
		return
	}
	for _, row := range rows {
		if err := w.syncOneLink(ctx, q, row); err != nil {
			slog.Warn("platformsync: sync link", "campaign_id", uuidString(row.CampaignID), "network", row.Network, "error", err)
		}
	}
}

func (w *Worker) syncOneLink(ctx context.Context, q *db.Queries, row db.PlatformCampaignLink) error {
	cred, credRow, err := w.loadCredential(ctx, row.CustomerID, row.Network)
	if err != nil {
		return w.markSyncError(ctx, q, row, err)
	}
	if w.costWorker != nil {
		if refreshErr := w.costWorker.MaybeRefreshToken(ctx, row.Network, credRow, &cred); refreshErr != nil {
			return w.markSyncError(ctx, q, row, refreshErr)
		}
	}

	remote, fetchErr := w.fetchRemoteStatus(ctx, row.Network, cred, row.ExternalCampaignID)
	if fetchErr != nil {
		return w.markSyncError(ctx, q, row, fetchErr)
	}

	var budgetMicro pgtype.Int8
	if remote.HasDailyBudgetMicro {
		budgetMicro = pgtype.Int8{Int64: remote.DailyBudgetMicro, Valid: true}
	}
	return q.UpdatePlatformCampaignLinkStatus(ctx, db.UpdatePlatformCampaignLinkStatusParams{
		CampaignID:               row.CampaignID,
		Network:                  row.Network,
		ExternalStatus:           remote.Status,
		ExternalBudgetResource:   remote.BudgetResource,
		ExternalDailyBudgetMicro: budgetMicro,
		SyncError:                pgtype.Text{},
	})
}

func (w *Worker) markSyncError(ctx context.Context, q *db.Queries, row db.PlatformCampaignLink, syncErr error) error {
	msg := syncErr.Error()
	return q.UpdatePlatformCampaignLinkStatus(ctx, db.UpdatePlatformCampaignLinkStatusParams{
		CampaignID:               row.CampaignID,
		Network:                  row.Network,
		ExternalStatus:           row.ExternalStatus,
		ExternalBudgetResource:   row.ExternalBudgetResource,
		ExternalDailyBudgetMicro: row.ExternalDailyBudgetMicro,
		SyncError:                pgtype.Text{String: msg, Valid: true},
	})
}

func (w *Worker) fetchRemoteStatus(ctx context.Context, network string, cred costsync.Credential, externalCampaignID string) (RemoteCampaignStatus, error) {
	switch NormalizeNetwork(network) {
	case NetworkFacebook:
		return fetchFacebookCampaignStatus(ctx, w.httpClient, w.networkBase[NetworkFacebook], cred, externalCampaignID)
	case NetworkGoogle:
		return fetchGoogleCampaignStatus(ctx, w.httpClient, w.networkBase[NetworkGoogle], cred, externalCampaignID)
	default:
		return RemoteCampaignStatus{}, fmt.Errorf("unsupported network %q", network)
	}
}

func (w *Worker) loadCredential(ctx context.Context, customerID pgtype.UUID, network string) (costsync.Credential, db.CostSyncCredential, error) {
	row, err := db.New(w.pool).GetCostSyncCredential(ctx, db.GetCostSyncCredentialParams{
		CustomerID: customerID,
		Network:    network,
	})
	if err != nil {
		return costsync.Credential{}, db.CostSyncCredential{}, fmt.Errorf("cost sync credential: %w", err)
	}
	if w.costWorker == nil {
		return costsync.Credential{}, row, fmt.Errorf("platformsync: cost worker not configured")
	}
	cred, err := w.costWorker.DecryptCredential(row)
	if err != nil {
		return costsync.Credential{}, row, err
	}
	return cred, row, nil
}

func (w *Worker) processPendingMutations(ctx context.Context) {
	q := db.New(w.pool)
	rows, err := q.ListPendingPlatformCampaignMutations(ctx, int32(syncBatchSize))
	if err != nil {
		slog.Warn("platformsync: list pending mutations", "error", err)
		return
	}
	for _, row := range rows {
		if err := w.applyMutation(ctx, q, row); err != nil {
			slog.Warn("platformsync: apply mutation", "id", row.ID, "error", err)
		}
	}
}

func (w *Worker) ApplyMutationByKey(ctx context.Context, idempotencyKey string) (db.PlatformCampaignMutation, error) {
	q := db.New(w.pool)
	row, err := q.GetPlatformCampaignMutationByKey(ctx, idempotencyKey)
	if err != nil {
		return db.PlatformCampaignMutation{}, err
	}
	if row.Status == MutationPending {
		if applyErr := w.applyMutation(ctx, q, row); applyErr != nil {
			return db.PlatformCampaignMutation{}, applyErr
		}
		return q.GetPlatformCampaignMutationByKey(ctx, idempotencyKey)
	}
	return row, nil
}

func (w *Worker) applyMutation(ctx context.Context, q *db.Queries, row db.PlatformCampaignMutation) error {
	link, err := q.GetPlatformCampaignLink(ctx, db.GetPlatformCampaignLinkParams{
		CampaignID: row.CampaignID,
		Network:    row.Network,
	})
	if err != nil {
		return w.completeMutation(ctx, q, row.ID, MutationFailed, nil, err)
	}

	cred, credRow, err := w.loadCredential(ctx, row.CustomerID, row.Network)
	if err != nil {
		return w.completeMutation(ctx, q, row.ID, MutationFailed, nil, err)
	}
	if w.costWorker != nil {
		if refreshErr := w.costWorker.MaybeRefreshToken(ctx, row.Network, credRow, &cred); refreshErr != nil {
			return w.completeMutation(ctx, q, row.ID, MutationFailed, nil, refreshErr)
		}
	}

	var req MutationRequest
	if len(row.RequestJson) > 0 {
		_ = json.Unmarshal(row.RequestJson, &req)
	}

	resp, applyErr := w.mutateRemote(ctx, row.Network, cred, link, row.Action, req)
	if applyErr != nil {
		return w.completeMutation(ctx, q, row.ID, MutationFailed, resp, applyErr)
	}
	if syncErr := w.syncOneLink(ctx, q, link); syncErr != nil {
		slog.Warn("platformsync: post-mutation sync", "error", syncErr)
	}
	return w.completeMutation(ctx, q, row.ID, MutationApplied, resp, nil)
}

func (w *Worker) completeMutation(ctx context.Context, q *db.Queries, id int64, status string, resp any, applyErr error) error {
	var respJSON []byte
	if resp != nil {
		respJSON, _ = json.Marshal(resp)
	}
	var errMsg pgtype.Text
	if applyErr != nil {
		errMsg = pgtype.Text{String: applyErr.Error(), Valid: true}
	}
	return q.CompletePlatformCampaignMutation(ctx, db.CompletePlatformCampaignMutationParams{
		ID:           id,
		Status:       status,
		ResponseJson: respJSON,
		ErrorMessage: errMsg,
	})
}

func (w *Worker) mutateRemote(ctx context.Context, network string, cred costsync.Credential, link db.PlatformCampaignLink, action string, req MutationRequest) (map[string]string, error) {
	switch NormalizeNetwork(network) {
	case NetworkFacebook:
		return mutateFacebookCampaign(ctx, w.httpClient, w.networkBase[NetworkFacebook], cred, link.ExternalCampaignID, action, req)
	case NetworkGoogle:
		return mutateGoogleCampaign(ctx, w.httpClient, w.networkBase[NetworkGoogle], cred, link, action, req)
	default:
		return nil, fmt.Errorf("unsupported network %q", network)
	}
}

func uuidString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return uuid.UUID(id.Bytes).String()
}
