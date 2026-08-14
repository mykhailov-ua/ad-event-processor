package controlplane

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/internal/notify"
	"github.com/bidshard/ad-event-processor/pkg/branding"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	defaultOpsMetricScrapeInterval = 15 * time.Second
	defaultOpsMetricRetention      = 24 * time.Hour
	defaultOpsMetricScrapeTimeout  = 5 * time.Second
)

const insertOpsMetricSampleSQL = `INSERT INTO ops.metric_samples (name, labels_hash, ts, value)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (name, labels_hash, ts) DO UPDATE SET value = EXCLUDED.value`

type opsReader struct {
	svc *Service
}

func newOpsReader(svc *Service) *opsReader {
	if svc == nil {
		return nil
	}
	return &opsReader{svc: svc}
}

func (r *opsReader) GetIncidentSnapshot(ctx context.Context) (adminapi.IncidentSnapshotDTO, error) {
	report, err := r.svc.GetShardHealth(ctx)
	if err != nil {
		return adminapi.IncidentSnapshotDTO{}, err
	}
	return adminapi.IncidentSnapshotDTO{
		EmergencyBreaker: report.EmergencyBreaker,
		Shards:           report.Shards,
		Outbox:           report.Outbox,
		StreamLag:        []adminapi.ShardStreamLag{},
		BreakerStates:    map[string]string{},
	}, nil
}

func (r *opsReader) ListOutboxEvents(ctx context.Context, status, eventType, cursor string, limit int32) (adminapi.OutboxListResult, error) {
	if r.svc.GetPool() == nil {
		return adminapi.OutboxListResult{}, fmt.Errorf("postgres pool not configured")
	}
	if limit <= 0 {
		limit = 50
	}
	var cursorID int64
	if cursor != "" {
		n, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			return adminapi.OutboxListResult{}, errInvalidQuery("invalid cursor")
		}
		cursorID = n
	}
	rows, err := r.svc.GetPool().Query(ctx, `
		SELECT id, event_type, status, created_at
		FROM outbox_events
		WHERE ($1::text = '' OR status = $1)
		  AND ($2::text = '' OR event_type = $2)
		  AND ($3::bigint = 0 OR id < $3)
		ORDER BY id DESC
		LIMIT $4`, status, eventType, cursorID, limit+1)
	if err != nil {
		return adminapi.OutboxListResult{}, err
	}
	defer rows.Close()

	var items []adminapi.OutboxEventDTO
	for rows.Next() {
		var id int64
		var eventTypeVal, statusVal string
		var createdAt time.Time
		if err := rows.Scan(&id, &eventTypeVal, &statusVal, &createdAt); err != nil {
			return adminapi.OutboxListResult{}, err
		}
		items = append(items, adminapi.OutboxEventDTO{
			ID:        id,
			EventType: eventTypeVal,
			Status:    statusVal,
			CreatedAt: createdAt.UTC().Format(time.RFC3339),
		})
	}
	result := adminapi.OutboxListResult{Items: items, Total: int64(len(items))}
	if int32(len(items)) > limit {
		result.Items = items[:limit]
		result.NextCursor = strconv.FormatInt(result.Items[len(result.Items)-1].ID, 10)
	}
	return result, rows.Err()
}

func (r *opsReader) ListDLQEntries(ctx context.Context, cursor string, limit int) (adminapi.FanOutResult[adminapi.DLQEntryDTO], error) {
	return r.listDLQEntries(ctx, cursor, limit)
}

func (r *opsReader) EnqueueDLQRetry(ctx context.Context, payload adminapi.DLQRetryPayload, idempotencyKey string) error {
	return r.enqueueDLQRetry(ctx, payload, idempotencyKey)
}

func (r *opsReader) GetShardHealthFanOut(ctx context.Context) (adminapi.ShardHealthAPIResponse, error) {
	report, err := r.svc.GetShardHealth(ctx)
	if err != nil {
		return adminapi.ShardHealthAPIResponse{}, err
	}
	return adminapi.ShardHealthAPIResponse{
		ShardHealthReport: report,
	}, nil
}

func (r *opsReader) ExportAuditCSV(ctx context.Context, cursor string, redactPII bool, w io.Writer) (adminapi.AuditExportResult, error) {
	var cursorID int64
	if cursor != "" {
		n, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			return adminapi.AuditExportResult{}, errInvalidQuery("invalid cursor")
		}
		cursorID = n
	}
	cw := csv.NewWriter(w)
	if cursorID == 0 {
		_ = cw.Write([]string{"id", "admin_id", "action", "target_type", "target_id", "is_masked", "created_at"})
	}
	rows, err := db.New(r.svc.GetPool()).ListAuditLogsExport(ctx, db.ListAuditLogsExportParams{
		Column1: cursorID,
		Limit:   500,
	})
	if err != nil {
		return adminapi.AuditExportResult{}, err
	}
	var lastID int64
	for _, row := range rows {
		adminID := ""
		if row.AdminID.Valid {
			adminID = uuid.UUID(row.AdminID.Bytes).String()
		}
		targetID := ""
		if row.TargetID.Valid {
			targetID = uuid.UUID(row.TargetID.Bytes).String()
		}
		createdAt := ""
		if row.CreatedAt.Valid {
			createdAt = row.CreatedAt.Time.UTC().Format(time.RFC3339)
		}
		if redactPII {
			if adminID != "" {
				adminID = "***"
			}
			if targetID != "" {
				targetID = "***"
			}
		}
		_ = cw.Write([]string{
			strconv.FormatInt(row.ID, 10),
			adminID,
			row.Action,
			row.TargetType,
			targetID,
			strconv.FormatBool(row.IsMasked),
			createdAt,
		})
		lastID = row.ID
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return adminapi.AuditExportResult{}, err
	}
	byteCount := 0
	if buf, ok := w.(*bytes.Buffer); ok {
		byteCount = buf.Len()
	}
	result := adminapi.AuditExportResult{Bytes: byteCount}
	if len(rows) >= 500 {
		result.Truncated = true
		result.NextCursor = strconv.FormatInt(lastID, 10)
	}
	return result, nil
}

func (r *opsReader) LookupLedgerIDForPaymentIntent(ctx context.Context, intentID string) (string, error) {
	paymentIntentID, err := uuid.Parse(intentID)
	if err != nil {
		return "", err
	}
	row, err := db.New(r.svc.GetPool()).GetLedgerByPaymentIntent(ctx, domain.ToUUID(paymentIntentID))
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(row.ID, 10), nil
}

func (r *opsReader) ListReconRuns(ctx context.Context, service string, limit, offset int32) ([]adminapi.ReconRunDTO, int64, error) {
	return r.svc.ListReconRuns(ctx, service, limit, offset)
}

type OpsAlerter struct {
	client             *NotifierClient
	provider           string
	recipient          string
	broadcastProviders []string
	cooldown           time.Duration
	outboxStuckSec     int
	lastSent           sync.Map
	enqueueFailures    atomic.Int64
	wg                 sync.WaitGroup
}

func NewOpsAlerter(client *NotifierClient, cfg *config.Config) *OpsAlerter {
	if client == nil || cfg == nil || !cfg.OpsAlertsEnabled() {
		return nil
	}
	provider, recipient, ok := resolveOpsAlertTarget(cfg)
	if !ok {
		return nil
	}
	cooldownSec := cfg.Management.OpsAlertCooldownSec
	if cooldownSec <= 0 {
		cooldownSec = 300
	}
	outboxStuckSec := cfg.Management.OpsAlertOutboxStuckSec
	if outboxStuckSec <= 0 {
		outboxStuckSec = 120
	}
	return &OpsAlerter{
		client:             client,
		provider:           provider,
		recipient:          recipient,
		broadcastProviders: resolveBroadcastProviders(cfg),
		cooldown:           time.Duration(cooldownSec) * time.Second,
		outboxStuckSec:     outboxStuckSec,
	}
}

func (a *OpsAlerter) shouldSend(key string) bool {
	if a == nil {
		return false
	}
	now := time.Now()
	if v, ok := a.lastSent.Load(key); ok {
		if now.Sub(v.(time.Time)) < a.cooldown {
			return false
		}
	}
	a.lastSent.Store(key, now)
	return true
}

func (a *OpsAlerter) sendAsync(ctx context.Context, key, title, body string, broadcast bool) {
	if a == nil || !a.shouldSend(key) {
		return
	}
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := a.dispatch(sendCtx, key, title, body, broadcast); err != nil {
			failures := a.enqueueFailures.Add(1)
			metrics.IncControlOpsAlertEnqueueFailures()
			slog.Warn("ops alert enqueue failed", "key", key, "error", err, "consecutive_failures", failures)
			if failures == 1 || failures%5 == 0 {
				metaTitle := branding.AlertTitle("ops alert enqueue failing")
				metaBody := fmt.Sprintf(
					"<b>Notifier enqueue failures</b>\nConsecutive failures: %d\nLast key: %s\nError: %v",
					failures, key, err,
				)
				if a.shouldSend("notifier:enqueue") {
					if metaErr := a.dispatch(sendCtx, "notifier:enqueue", metaTitle, metaBody, true); metaErr != nil {
						slog.Warn("ops meta alert enqueue failed", "error", metaErr)
					}
				}
			}
			return
		}
		a.enqueueFailures.Store(0)
	}()
}

func (a *OpsAlerter) Drain() {
	if a != nil {
		a.wg.Wait()
	}
}

func (a *OpsAlerter) dispatch(ctx context.Context, key, title, body string, broadcast bool) error {
	if a == nil {
		return fmt.Errorf("ops alerter not configured")
	}
	target := notify.OpsAlertTarget{Provider: a.provider, Recipient: a.recipient}
	return enqueueOpsNotification(ctx, a.client, target, title, body, key, broadcast, a.broadcastProviders)
}

func (a *OpsAlerter) enqueueNotification(ctx context.Context, key, title, body string, broadcast bool) error {
	return a.dispatch(ctx, key, title, body, broadcast)
}

func (a *OpsAlerter) AlertReconDiscrepancy(ctx context.Context, runID int64, discrepancies int, totalDelta int64, period string) {
	if a == nil || discrepancies <= 0 {
		return
	}
	key := fmt.Sprintf("recon:run:%d", runID)
	title := branding.AlertTitle("recon discrepancy")
	body := fmt.Sprintf(
		"<b>Recon discrepancy</b>\nPeriod: %s\nRun #%d\nCampaigns adjusted: %d\nTotal delta (micro): %d",
		period, runID, discrepancies, totalDelta,
	)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) AlertReconDiscrepancyUnresolved(ctx context.Context, runID int64, unresolved int, totalDelta int64, period string, oldest time.Time) {
	if a == nil || unresolved <= 0 {
		return
	}
	key := fmt.Sprintf("recon:unresolved:%d", runID)
	title := branding.AlertTitle("unreconciled budget drift")
	body := fmt.Sprintf(
		"<b>Unresolved recon discrepancy</b>\nPeriod: %s\nRun #%d\nUnresolved campaigns: %d\nTotal |delta| (micro): %d\nOldest since: %s",
		period, runID, unresolved, totalDelta, oldest.UTC().Format(time.RFC3339),
	)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) AlertRedisShardUnhealthy(ctx context.Context, shardIdx int, err error) {
	if a == nil {
		return
	}
	key := fmt.Sprintf("redis:shard:%d", shardIdx)
	title := fmt.Sprintf("%s: Redis shard %d unreachable", branding.ProductName(), shardIdx)
	body := fmt.Sprintf(
		"<b>Redis shard unhealthy</b>\nShard: %d\nError: %v\nStuck quota reservations were released.",
		shardIdx, err,
	)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) AlertSlotMapMigrating(ctx context.Context, version int32, slots []int16, targetShard int16) {
	if a == nil || len(slots) == 0 {
		return
	}
	key := fmt.Sprintf("migration:mark:%d:%d:%s", version, targetShard, formatSlotIDs(slots))
	title := branding.AlertTitle("slot map migration started")
	body := fmt.Sprintf(
		"<b>Slot map migration</b>\nVersion: %d\nTarget shard: %d\nSlots (%d): %s\nNext: copy data, then activate.",
		version, targetShard, len(slots), formatSlotIDs(slots),
	)
	a.sendAsync(ctx, key, title, body, false)
}

func (a *OpsAlerter) AlertDrainStuck(ctx context.Context, version int32, slot int16, state, lastError string, updatedAt time.Time) {
	if a == nil {
		return
	}
	key := fmt.Sprintf("drain:%d:%d:%s", version, slot, state)
	title := branding.AlertTitle("slot migration drain stuck")
	body := fmt.Sprintf(
		"<b>Drain stuck</b>\nVersion: %d\nSlot: %d\nState: %s\nSince: %s\nError: %s",
		version, slot, state, updatedAt.UTC().Format(time.RFC3339), lastError,
	)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) AlertBlacklistJanitorFailed(ctx context.Context, err error) {
	if a == nil || err == nil {
		return
	}
	key := "blacklist:janitor:scan"
	title := branding.AlertTitle("blacklist janitor failed")
	body := fmt.Sprintf("<b>Blacklist janitor scan failed</b>\nError: %v", err)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) AlertOutboxStuck(ctx context.Context, pending int64, oldestSeconds float64) {
	if a == nil || pending <= 0 {
		return
	}
	key := fmt.Sprintf("outbox:stuck:%d", int64(oldestSeconds)/60)
	title := branding.AlertTitle("outbox backlog stale")
	body := fmt.Sprintf(
		"<b>Outbox backlog stale</b>\nPending events: %d\nOldest pending age (s): %.0f\nHot-path Redis may drift from Postgres.",
		pending, oldestSeconds,
	)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) OutboxStuckThresholdSec() int {
	if a == nil || a.outboxStuckSec <= 0 {
		return 120
	}
	return a.outboxStuckSec
}

func (a *OpsAlerter) AlertSlotMigrationComplete(ctx context.Context, version int32) {
	if a == nil {
		return
	}
	key := fmt.Sprintf("migration:complete:%d", version)
	title := branding.AlertTitle("slot migration cutover complete")
	body := fmt.Sprintf("<b>Slot migration complete</b>\nActive version: %d\nPost-cutover R5 verification passed.", version)
	a.sendAsync(ctx, key, title, body, false)
}

func (a *OpsAlerter) AlertLedgerDrift(ctx context.Context, customerID string, driftErr error) {
	if a == nil || driftErr == nil {
		return
	}
	key := fmt.Sprintf("billing:drift:%s", customerID)
	title := branding.AlertTitle("billing ledger drift")
	body := fmt.Sprintf("<b>Ledger invariant failed</b>\nCustomer: %s\nError: %v", customerID, driftErr)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) AlertCHEmergencyDrop(ctx context.Context, table, partition string, diskUsedPct float64, thresholdPct int) {
	if a == nil {
		return
	}
	key := fmt.Sprintf("ch:emergency:%s:%s", table, partition)
	title := branding.AlertTitle("ClickHouse emergency partition drop")
	body := fmt.Sprintf(
		"<b>CH emergency drop</b>\nTable: %s\nPartition: %s\nDisk used: %.1f%%\nThreshold: %d%%\nReview retention and ingest volume.",
		table, partition, diskUsedPct, thresholdPct,
	)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) AlertSlotMigrationError(ctx context.Context, stage string, err error) {
	if a == nil || err == nil {
		return
	}
	key := fmt.Sprintf("migration:tick:%s", stage)
	title := fmt.Sprintf("%s: slot migration %s failed", branding.ProductName(), stage)
	body := fmt.Sprintf("<b>Slot migration error</b>\nStage: %s\nError: %v", stage, err)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) AlertLicenseApplied(ctx context.Context, deploymentID string, validUntil time.Time, adminID string, revoked bool) {
	if a == nil {
		return
	}
	key := "license:apply:" + deploymentID
	title := fmt.Sprintf("%s: license applied", branding.ProductName())
	kind := "renewal"
	if revoked {
		kind = "revocation"
	}
	body := fmt.Sprintf(
		"<b>License %s</b>\nDeployment: %s\nValid until: %s\nAdmin: %s",
		kind, deploymentID, validUntil.UTC().Format(time.RFC3339), adminID,
	)
	a.sendAsync(ctx, key, title, body, false)
}

func formatSlotIDs(slots []int16) string {
	if len(slots) == 0 {
		return ""
	}
	const maxShown = 12
	parts := make([]string, 0, min(len(slots), maxShown))
	for i, slot := range slots {
		if i >= maxShown {
			parts = append(parts, fmt.Sprintf("+%d more", len(slots)-maxShown))
			break
		}
		parts = append(parts, strconv.Itoa(int(slot)))
	}
	return strings.Join(parts, ", ")
}

type OpsMetricScraper struct {
	svc       *Service
	pool      *pgxpool.Pool
	url       string
	client    *http.Client
	interval  time.Duration
	retention time.Duration
	fetch     func(ctx context.Context, url string) ([]byte, string, error)
}

func NewOpsMetricScraper(svc *Service, scrapeURL string) *OpsMetricScraper {
	if scrapeURL == "" {
		scrapeURL = "http://127.0.0.1:8188/metrics"
	}
	client := &http.Client{Timeout: defaultOpsMetricScrapeTimeout}
	w := &OpsMetricScraper{
		svc:       svc,
		interval:  defaultOpsMetricScrapeInterval,
		retention: defaultOpsMetricRetention,
		client:    client,
		url:       scrapeURL,
	}
	if svc != nil {
		w.pool = svc.GetPool()
	}
	w.fetch = func(ctx context.Context, url string) ([]byte, string, error) {
		return fetchMetrics(ctx, client, url)
	}
	return w
}

func fetchMetrics(ctx context.Context, client *http.Client, url string) ([]byte, string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("metrics scrape status %d", resp.StatusCode)
	}
	return body, resp.Header.Get("Content-Type"), nil
}

func (s *Service) StartOpsMetricScraper(ctx context.Context, scrapeURL string) {
	if s == nil || s.GetPool() == nil {
		return
	}
	w := NewOpsMetricScraper(s, scrapeURL)
	s.StartBackgroundWorker(func() {
		w.Start(ctx)
	})
}

func (w *OpsMetricScraper) Start(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	slog.Info("ops metric scraper starting", "url", w.url, "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	w.tick(ctx, time.Now().UTC())
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			w.tick(ctx, t.UTC())
		}
	}
}

func (w *OpsMetricScraper) tick(ctx context.Context, now time.Time) {
	run := func(runCtx context.Context) error {
		if err := w.scrapeAndStore(runCtx, now); err != nil {
			return err
		}
		return w.expireSamples(runCtx, now)
	}
	if w.svc != nil {
		if err := w.svc.withPgLow(ctx, run); err != nil {
			slog.Error("ops metric scraper tick failed", "error", err)
		}
		return
	}
	if err := run(ctx); err != nil {
		slog.Error("ops metric scraper tick failed", "error", err)
	}
}

func (w *OpsMetricScraper) scrapeAndStore(ctx context.Context, now time.Time) error {
	fetch := w.fetch
	if fetch == nil {
		fetch = func(ctx context.Context, url string) ([]byte, string, error) {
			return fetchMetrics(ctx, w.client, url)
		}
	}
	body, contentType, err := fetch(ctx, w.url)
	if err != nil {
		return fmt.Errorf("fetch metrics: %w", err)
	}
	samples, err := parsePrometheusMetrics(bytes.NewReader(body), contentType)
	if err != nil {
		return fmt.Errorf("parse metrics: %w", err)
	}
	if len(samples) == 0 {
		return nil
	}

	const insertSQL = insertOpsMetricSampleSQL
	ts := pgtype.Timestamptz{Time: now, Valid: true}
	batch := &pgx.Batch{}
	for _, sample := range samples {
		batch.Queue(insertSQL, sample.Name, sample.LabelsHash, ts, sample.Value)
	}
	br := w.pool.SendBatch(ctx, batch)
	var batchErr error
	for range samples {
		if _, err := br.Exec(); err != nil && batchErr == nil {
			batchErr = fmt.Errorf("insert metric sample batch: %w", err)
		}
	}
	if closeErr := br.Close(); closeErr != nil && batchErr == nil {
		batchErr = fmt.Errorf("close metric batch: %w", closeErr)
	}
	return batchErr
}

func (w *OpsMetricScraper) expireSamples(ctx context.Context, now time.Time) error {
	cutoff := now.Add(-w.retention)
	q := db.New(w.pool)
	if _, err := q.DeleteExpiredOpsMetricSamples(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true}); err != nil {
		return fmt.Errorf("expire metric samples: %w", err)
	}
	return nil
}

const dashboardMetricsBucketSec = 300

const mlManualLabelsListLimit = 100

func (r *opsReader) GetDashboardSummary(ctx context.Context) (adminapi.DashboardSummaryDTO, error) {
	if r == nil || r.svc == nil {
		return adminapi.DashboardSummaryDTO{}, fmt.Errorf("service not configured")
	}
	now := time.Now().UTC()
	snap, err := r.GetIncidentSnapshot(ctx)
	if err != nil {
		return adminapi.DashboardSummaryDTO{}, err
	}
	services := buildDashboardTopology(ctx, r.svc, snap)
	driftMax, rps, err := r.readDashboardLiveSignals(ctx, now)
	if err != nil {
		return adminapi.DashboardSummaryDTO{}, err
	}
	return adminapi.DashboardSummaryDTO{
		GeneratedAt:      now.Format(time.RFC3339),
		Services:         services,
		DriftMicroMax:    driftMax,
		DriftAlert:       driftMax > 0,
		RPSEstimate:      rps,
		OutboxPending:    snap.Outbox.Pending,
		EmergencyBreaker: snap.EmergencyBreaker,
	}, nil
}

func (r *opsReader) GetDashboardMetrics(ctx context.Context, rangeHours int, metricName string) (adminapi.DashboardMetricsDTO, error) {
	if r == nil || r.svc == nil || r.svc.GetPool() == nil {
		return adminapi.DashboardMetricsDTO{}, fmt.Errorf("postgres pool not configured")
	}
	if rangeHours <= 0 {
		rangeHours = 24
	}
	if rangeHours > 24 {
		rangeHours = 24
	}
	now := time.Now().UTC()
	since := now.Add(-time.Duration(rangeHours) * time.Hour)
	q := db.New(r.svc.GetPool())
	rows, err := q.ListOpsMetricSamplesDownsampled(ctx, db.ListOpsMetricSamplesDownsampledParams{
		Ts:      pgtype.Timestamptz{Time: since, Valid: true},
		Ts_2:    pgtype.Timestamptz{Time: now, Valid: true},
		Column3: metricName,
		Column4: float64(dashboardMetricsBucketSec),
	})
	if err != nil {
		return adminapi.DashboardMetricsDTO{}, err
	}
	points := make([]adminapi.DashboardMetricPoint, 0, len(rows))
	for _, row := range rows {
		ts, ok := metricSampleTime(row.Ts)
		if !ok {
			continue
		}
		points = append(points, adminapi.DashboardMetricPoint{
			Name:       row.Name,
			LabelsHash: row.LabelsHash,
			Timestamp:  ts.UTC().Format(time.RFC3339),
			Value:      row.Value,
		})
	}
	return adminapi.DashboardMetricsDTO{
		Range:       fmt.Sprintf("%dh", rangeHours),
		BucketSec:   dashboardMetricsBucketSec,
		Points:      points,
		GeneratedAt: now.Format(time.RFC3339),
	}, nil
}

func buildDashboardTopology(ctx context.Context, svc *Service, snap adminapi.IncidentSnapshotDTO) []adminapi.DashboardServiceCard {
	cards := []adminapi.DashboardServiceCard{
		{ID: "management", Name: "Management", Status: "ok"},
		{ID: "tracker", Name: "Tracker", Status: "unknown"},
		{ID: "processor", Name: "Processor", Status: "unknown"},
	}
	if svc != nil && svc.GetPool() != nil {
		status := "ok"
		detail := ""
		if err := svc.GetPool().Ping(ctx); err != nil {
			status = "down"
			detail = err.Error()
		}
		cards = append(cards, adminapi.DashboardServiceCard{ID: "pg", Name: "Postgres", Status: status, Detail: detail})
	} else {
		cards = append(cards, adminapi.DashboardServiceCard{ID: "pg", Name: "Postgres", Status: "down"})
	}
	chStatus := "disabled"
	if svc != nil && svc.cfg != nil && svc.cfg.ClickHouseEnabled() {
		chStatus = "ok"
		if svc.CHQuery() == nil {
			chStatus = "down"
		}
	}
	cards = append(cards, adminapi.DashboardServiceCard{ID: "ch", Name: "ClickHouse", Status: chStatus})
	for _, shard := range snap.Shards {
		status := "ok"
		if !shard.PingOK {
			status = "down"
		}
		cards = append(cards, adminapi.DashboardServiceCard{
			ID:     fmt.Sprintf("redis-%d", shard.ShardID),
			Name:   fmt.Sprintf("Redis %d", shard.ShardID),
			Status: status,
			Detail: shard.PingError,
		})
	}
	if snap.Outbox.Pending > 0 {
		for i := range cards {
			if cards[i].ID == "processor" {
				cards[i].Status = "degraded"
				cards[i].Detail = fmt.Sprintf("outbox_pending=%d", snap.Outbox.Pending)
			}
		}
	} else {
		for i := range cards {
			if cards[i].ID == "processor" {
				cards[i].Status = "ok"
			}
		}
	}
	return cards
}

func (r *opsReader) readDashboardLiveSignals(ctx context.Context, now time.Time) (driftMax float64, rps float64, err error) {
	pool := r.svc.GetPool()
	if pool == nil {
		return 0, 0, nil
	}
	q := db.New(pool)
	driftRow, derr := q.GetLatestOpsMetricSample(ctx, db.GetLatestOpsMetricSampleParams{
		Name:       "ad_recon_drift_micro_max",
		LabelsHash: "",
	})
	if derr == nil {
		driftMax = driftRow.Value
	}
	prevSince := now.Add(-2 * defaultOpsMetricScrapeInterval)
	rows, qerr := q.ListOpsMetricSamplesWindow(ctx, db.ListOpsMetricSamplesWindowParams{
		Ts:      pgtype.Timestamptz{Time: prevSince, Valid: true},
		Ts_2:    pgtype.Timestamptz{Time: now, Valid: true},
		Column3: "ad_http_requests_total",
	})
	if qerr != nil || len(rows) < 2 {
		return driftMax, 0, nil
	}
	first := rows[0]
	last := rows[len(rows)-1]
	if !first.Ts.Valid || !last.Ts.Valid {
		return driftMax, 0, nil
	}
	delta := last.Value - first.Value
	secs := last.Ts.Time.Sub(first.Ts.Time).Seconds()
	if secs > 0 && delta >= 0 {
		rps = delta / secs
	}
	return driftMax, rps, nil
}

func metricSampleTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, true
	case pgtype.Timestamptz:
		if t.Valid {
			return t.Time, true
		}
	}
	return time.Time{}, false
}

func (r *opsReader) GetMLModelStatus(ctx context.Context) (adminapi.MLModelStatusDTO, error) {
	if r == nil || r.svc == nil || r.svc.GetPool() == nil {
		return adminapi.MLModelStatusDTO{}, fmt.Errorf("postgres pool not configured")
	}

	status := adminapi.MLModelStatusDTO{
		ShardSync: []adminapi.MLShardSyncDTO{},
	}

	active, err := r.loadMLModelVersion(ctx, "ACTIVE")
	if err != nil {
		return adminapi.MLModelStatusDTO{}, err
	}
	if active != nil {
		status.ActiveVersion = active
		status.Importance = topFeatureImportance(active.ArtifactMetadata, 5)
	}

	syncing, err := r.loadMLModelVersion(ctx, "SYNCING")
	if err != nil {
		return adminapi.MLModelStatusDTO{}, err
	}
	if syncing != nil {
		status.SyncingVersion = syncing
		if len(status.Importance) == 0 {
			status.Importance = topFeatureImportance(syncing.ArtifactMetadata, 5)
		}
	}

	shardSync, err := r.loadMLShardSyncState(ctx)
	if err != nil {
		return adminapi.MLModelStatusDTO{}, err
	}
	status.ShardSync = shardSync

	redisStatus, err := r.readMLModelRedis(ctx)
	if err != nil {
		return adminapi.MLModelStatusDTO{}, err
	}
	status.Redis = redisStatus

	if evalReport, err := r.readMLDriftReport(); err == nil {
		status.Drift = evalReport.Drift
		status.DriftDetected = evalReport.DriftDetected
		status.Precision = evalReport.Precision
		status.Recall = evalReport.Recall
	}

	return status, nil
}

type mlEvalReport struct {
	Precision     float64
	Recall        float64
	DriftDetected bool
	Drift         json.RawMessage
}

func (r *opsReader) readMLDriftReport() (mlEvalReport, error) {
	path := os.Getenv("FRAUD_EVAL_REPORT_PATH")
	if path == "" {
		path = "var/fraudscore/shadow_eval_report.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return mlEvalReport{}, err
	}
	var raw struct {
		Precision float64         `json:"precision"`
		Recall    float64         `json:"recall"`
		Drift     json.RawMessage `json:"drift"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return mlEvalReport{}, err
	}
	out := mlEvalReport{
		Precision: raw.Precision,
		Recall:    raw.Recall,
		Drift:     raw.Drift,
	}
	if len(raw.Drift) > 0 {
		var driftBlock struct {
			DriftDetected bool `json:"drift_detected"`
		}
		if json.Unmarshal(raw.Drift, &driftBlock) == nil {
			out.DriftDetected = driftBlock.DriftDetected
		}
	}
	return out, nil
}

func topFeatureImportance(metadata []byte, limit int) []adminapi.MLFeatureImportanceDTO {
	if len(metadata) == 0 || limit <= 0 {
		return nil
	}
	var meta struct {
		Importance map[string]float64 `json:"importance"`
	}
	if err := json.Unmarshal(metadata, &meta); err != nil || len(meta.Importance) == 0 {
		return nil
	}
	names := make([]string, 0, len(meta.Importance))
	for name := range meta.Importance {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return meta.Importance[names[i]] > meta.Importance[names[j]]
	})
	if len(names) > limit {
		names = names[:limit]
	}
	out := make([]adminapi.MLFeatureImportanceDTO, len(names))
	for i, name := range names {
		out[i] = adminapi.MLFeatureImportanceDTO{
			Name:  name,
			Value: meta.Importance[name],
		}
	}
	return out
}

func (r *opsReader) AddMLManualLabel(ctx context.Context, ipHash string, label int, reason string) error {
	if r == nil || r.svc == nil || r.svc.GetPool() == nil {
		return fmt.Errorf("postgres pool not configured")
	}
	if ipHash == "" {
		return errValidation("ip_hash required")
	}
	if len(ipHash) != 32 {
		return errValidation("ip_hash must be 32 hex characters")
	}
	for _, c := range ipHash {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return errValidation("ip_hash must be 32 hex characters")
	}
	if label != 0 && label != 1 {
		return errValidation("label must be 0 or 1")
	}
	_, err := r.svc.GetPool().Exec(ctx, `
		INSERT INTO ml_manual_labels (ip_hash, label, reason, source, created_at)
		VALUES ($1, $2, $3, 'admin_ui', NOW())
		ON CONFLICT (ip_hash) DO UPDATE SET
			label = EXCLUDED.label,
			reason = EXCLUDED.reason,
			created_at = NOW()`,
		ipHash, label, reason)
	return err
}

func (r *opsReader) ListMLManualLabels(ctx context.Context) ([]adminapi.MLManualLabelDTO, error) {
	if r == nil || r.svc == nil || r.svc.GetPool() == nil {
		return nil, fmt.Errorf("postgres pool not configured")
	}
	rows, err := r.svc.GetPool().Query(ctx, `
		SELECT ip_hash, label, reason, source, created_at
		FROM ml_manual_labels
		ORDER BY created_at DESC
		LIMIT $1`, mlManualLabelsListLimit)
	if err != nil {
		return nil, fmt.Errorf("query ml_manual_labels: %w", err)
	}
	defer rows.Close()

	var out []adminapi.MLManualLabelDTO
	for rows.Next() {
		var row adminapi.MLManualLabelDTO
		var createdAt time.Time
		if err := rows.Scan(&row.IPHash, &row.Label, &row.Reason, &row.Source, &createdAt); err != nil {
			return nil, err
		}
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *opsReader) loadMLModelVersion(ctx context.Context, modelStatus string) (*adminapi.MLModelVersionDTO, error) {
	var id, artifactHash, status string
	var metricsJSON []byte
	var createdAt time.Time
	err := r.svc.GetPool().QueryRow(ctx, `
		SELECT id, artifact_hash, status, metrics_json, created_at
		FROM ml_model_versions
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT 1`, modelStatus).Scan(&id, &artifactHash, &status, &metricsJSON, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query ml_model_versions %s: %w", modelStatus, err)
	}
	return &adminapi.MLModelVersionDTO{
		ID:               id,
		ArtifactHash:     artifactHash,
		Status:           status,
		CreatedAt:        createdAt.UTC().Format(time.RFC3339),
		ArtifactMetadata: json.RawMessage(metricsJSON),
	}, nil
}

func (r *opsReader) loadMLShardSyncState(ctx context.Context) ([]adminapi.MLShardSyncDTO, error) {
	rows, err := r.svc.GetPool().Query(ctx, `
		SELECT shard_id, model_version, phase, started_at
		FROM ml_shard_sync_state
		ORDER BY shard_id, model_version`)
	if err != nil {
		return nil, fmt.Errorf("query ml_shard_sync_state: %w", err)
	}
	defer rows.Close()

	var out []adminapi.MLShardSyncDTO
	for rows.Next() {
		var shardID int
		var modelVersion, phase string
		var startedAt time.Time
		if err := rows.Scan(&shardID, &modelVersion, &phase, &startedAt); err != nil {
			return nil, err
		}
		out = append(out, adminapi.MLShardSyncDTO{
			ShardID:      shardID,
			ModelVersion: modelVersion,
			Phase:        phase,
			StartedAt:    startedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, rows.Err()
}

func (r *opsReader) readMLModelRedis(ctx context.Context) (adminapi.MLModelRedisDTO, error) {
	rdbs := r.svc.rdbs
	if len(rdbs) == 0 {
		return adminapi.MLModelRedisDTO{}, nil
	}

	type shardRedis struct {
		version   string
		hash      string
		appliedAt int64
	}

	var shards []shardRedis
	for _, rdb := range rdbs {
		if rdb == nil {
			continue
		}
		pipe := rdb.Pipeline()
		verCmd := pipe.Get(ctx, "ml:model:version")
		hashCmd := pipe.Get(ctx, "ml:model:hash")
		appliedCmd := pipe.Get(ctx, "ml:model:applied_at")
		if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
			return adminapi.MLModelRedisDTO{}, fmt.Errorf("ml model redis pipeline: %w", err)
		}
		version, err := verCmd.Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return adminapi.MLModelRedisDTO{}, fmt.Errorf("query ml:model:version: %w", err)
		}
		hash, err := hashCmd.Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return adminapi.MLModelRedisDTO{}, fmt.Errorf("query ml:model:hash: %w", err)
		}
		appliedRaw, err := appliedCmd.Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return adminapi.MLModelRedisDTO{}, fmt.Errorf("query ml:model:applied_at: %w", err)
		}
		var appliedAt int64
		if appliedRaw != "" {
			appliedAt, _ = strconv.ParseInt(appliedRaw, 10, 64)
		}
		shards = append(shards, shardRedis{
			version:   version,
			hash:      hash,
			appliedAt: appliedAt,
		})
	}

	if len(shards) == 0 {
		return adminapi.MLModelRedisDTO{}, nil
	}

	ref := shards[0]
	consistent := true
	for _, s := range shards[1:] {
		if s.version != ref.version || s.hash != ref.hash {
			consistent = false
			break
		}
	}

	out := adminapi.MLModelRedisDTO{
		VersionID:        ref.version,
		Hash:             ref.hash,
		ShardsReporting:  len(shards),
		ShardsConsistent: consistent,
	}
	if ref.appliedAt > 0 {
		out.AppliedAt = time.Unix(ref.appliedAt, 0).UTC().Format(time.RFC3339)
	}
	return out, nil
}
