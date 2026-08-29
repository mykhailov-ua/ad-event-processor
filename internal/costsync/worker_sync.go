package costsync

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"ad-event-processor/internal/costsync/provider"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	costSyncAdvisoryLockKey = int64(0x657370785f636f73)
	advisoryUnlockTimeout   = 5 * time.Second
)

type OAuthConfig struct {
	MetaAppID, MetaAppSecret                 string
	GoogleClientID, GoogleClientSecret       string
	TikTokAppID, TikTokAppSecret             string
	MicrosoftClientID, MicrosoftClientSecret string
	SnapchatClientID, SnapchatClientSecret   string
	SnapchatTokenURL                         string
	LinkedInClientID, LinkedInClientSecret   string
	LinkedInTokenURL                         string
	PinterestClientID, PinterestClientSecret string
	PinterestTokenURL                        string
}

type Worker struct {
	pool            *pgxpool.Pool
	converter       *CurrencyConverter
	encryptionKey   []byte
	httpClient      *http.Client
	networkBaseURL  map[string]string
	oauth           OAuthConfig
	insertSnapshots func(context.Context, []CostLine, []int64) error
	clickAttributor ClickCostAttributor
	onSyncComplete  func(network string, duration time.Duration)
	cycleWG         sync.WaitGroup
}

type WorkerOption func(*Worker)

func WithClickHouse(inserter *ClickHouseInserter) WorkerOption {
	return func(w *Worker) {
		if inserter != nil {
			w.insertSnapshots = inserter.InsertSnapshots
		}
	}
}

func WithClickAttributor(attr ClickCostAttributor) WorkerOption {
	return func(w *Worker) {
		if attr != nil {
			w.clickAttributor = attr
		}
	}
}

func WithMemorySnapshots(m *MemorySnapshotInserter) WorkerOption {
	return func(w *Worker) {
		if m != nil {
			w.insertSnapshots = m.InsertSnapshots
		}
	}
}

func WithOAuth(cfg OAuthConfig) WorkerOption {
	return func(w *Worker) {
		if cfg.MetaAppID != "" {
			w.oauth.MetaAppID = cfg.MetaAppID
			w.oauth.MetaAppSecret = cfg.MetaAppSecret
		}
		if cfg.GoogleClientID != "" {
			w.oauth.GoogleClientID = cfg.GoogleClientID
			w.oauth.GoogleClientSecret = cfg.GoogleClientSecret
		}
		if cfg.TikTokAppID != "" {
			w.oauth.TikTokAppID = cfg.TikTokAppID
			w.oauth.TikTokAppSecret = cfg.TikTokAppSecret
		}
		if cfg.MicrosoftClientID != "" {
			w.oauth.MicrosoftClientID = cfg.MicrosoftClientID
			w.oauth.MicrosoftClientSecret = cfg.MicrosoftClientSecret
		}
		if cfg.SnapchatClientID != "" {
			w.oauth.SnapchatClientID = cfg.SnapchatClientID
			w.oauth.SnapchatClientSecret = cfg.SnapchatClientSecret
			w.oauth.SnapchatTokenURL = cfg.SnapchatTokenURL
		}
		if cfg.LinkedInClientID != "" {
			w.oauth.LinkedInClientID = cfg.LinkedInClientID
			w.oauth.LinkedInClientSecret = cfg.LinkedInClientSecret
			w.oauth.LinkedInTokenURL = cfg.LinkedInTokenURL
		}
		if cfg.PinterestClientID != "" {
			w.oauth.PinterestClientID = cfg.PinterestClientID
			w.oauth.PinterestClientSecret = cfg.PinterestClientSecret
			w.oauth.PinterestTokenURL = cfg.PinterestTokenURL
		}
	}
}

func WithNetworkBaseURL(network, baseURL string) WorkerOption {
	return func(w *Worker) {
		w.networkBaseURL[network] = baseURL
	}
}

func WithSyncCompleteHook(fn func(network string, duration time.Duration)) WorkerOption {
	return func(w *Worker) {
		w.onSyncComplete = fn
	}
}

func NewWorker(pool *pgxpool.Pool, encryptionKey []byte, opts ...WorkerOption) *Worker {
	key := normalizeKey(encryptionKey)
	client := &http.Client{Timeout: 90 * time.Second}

	w := &Worker{
		pool:           pool,
		converter:      NewCurrencyConverter(pool, client),
		encryptionKey:  key,
		httpClient:     client,
		networkBaseURL: make(map[string]string),
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

func normalizeKey(key []byte) []byte {
	if len(key) == 0 {
		return []byte("postback-encryption-secret-key32")
	}
	if len(key) < 32 {
		padded := make([]byte, 32)
		copy(padded, key)
		return padded
	}
	if len(key) > 32 {
		return key[:32]
	}
	return key
}

func (w *Worker) Start(ctx context.Context) {
	slog.Info("cost-sync worker starting", "daily_interval", "1h", "subdaily_interval", "15m")
	hourly := time.NewTicker(time.Hour)
	subdaily := time.NewTicker(15 * time.Minute)
	defer hourly.Stop()
	defer subdaily.Stop()

	w.runHourlyGuarded(ctx, "cron")
	w.runSubdailyGuarded(ctx, "cron")

	for {
		select {
		case <-ctx.Done():
			w.Wait()
			return
		case <-hourly.C:
			w.runHourlyGuarded(ctx, "cron")
		case <-subdaily.C:
			w.runSubdailyGuarded(ctx, "cron")
		}
	}
}

func (w *Worker) runSubdailyGuarded(ctx context.Context, trigger string) {
	w.cycleWG.Add(1)
	go func() {
		defer w.cycleWG.Done()
		w.runSubdaily(ctx, trigger)
	}()
}

func (w *Worker) runSubdaily(ctx context.Context, trigger string) {
	opCtx, cancel := context.WithTimeout(ctx, 110*time.Second)
	defer cancel()

	acquired, err := w.tryAdvisoryLock(opCtx)
	if err != nil {
		slog.Error("cost-sync subdaily advisory lock failed", "error", err)
		return
	}
	if !acquired {
		slog.Debug("cost-sync subdaily skipped: another leader holds lock")
		return
	}
	defer w.releaseAdvisoryLock(opCtx)

	q := db.New(w.pool)
	creds, err := q.ListCostSyncCredentialsDueSubdaily(opCtx)
	if err != nil {
		slog.Error("cost-sync subdaily list failed", "error", err)
		return
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for i := range creds {
		credRow := creds[i]
		if err := w.syncCredentialSubdaily(opCtx, credRow, today, trigger); err != nil {
			slog.Warn("cost-sync subdaily network failed", "network", credRow.Network, "customer_id", credRow.CustomerID.Bytes, "error", err)
			metrics.CostSyncRunsTotal.WithLabelValues("failed").Inc()
		}
	}
}

func (w *Worker) runHourlyGuarded(ctx context.Context, trigger string) {
	w.cycleWG.Add(1)
	go func() {
		defer w.cycleWG.Done()
		w.runHourly(ctx, trigger)
	}()
}

func (w *Worker) Wait() {
	w.cycleWG.Wait()
}

func (w *Worker) RunManual(ctx context.Context, customerID *uuid.UUID, network string, from, to time.Time) error {
	if to.Before(from) {
		return fmt.Errorf("invalid date range")
	}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		if err := w.syncDay(ctx, customerID, network, d, "manual"); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) runHourly(ctx context.Context, trigger string) {
	opCtx, cancel := context.WithTimeout(ctx, 110*time.Second)
	defer cancel()

	acquired, err := w.tryAdvisoryLock(opCtx)
	if err != nil {
		slog.Error("cost-sync advisory lock failed", "error", err)
		return
	}
	if !acquired {
		slog.Debug("cost-sync skipped: another leader holds lock")
		return
	}
	defer w.releaseAdvisoryLock(opCtx)

	yesterday := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	if err := w.syncDay(opCtx, nil, "", yesterday, trigger); err != nil {
		slog.Error("cost-sync hourly run failed", "error", err)
		metrics.CostSyncRunsTotal.WithLabelValues("failed").Inc()
	}
}

func (w *Worker) syncDay(ctx context.Context, filterCustomer *uuid.UUID, filterNetwork string, date time.Time, trigger string) error {
	q := db.New(w.pool)
	creds, err := q.ListCostSyncCredentials(ctx)
	if err != nil {
		return err
	}

	for i := range creds {
		credRow := &creds[i]
		if filterCustomer != nil && credRow.CustomerID.Bytes != *filterCustomer {
			continue
		}
		if filterNetwork != "" && credRow.Network != filterNetwork {
			continue
		}
		if err := w.syncCredential(ctx, *credRow, date, trigger); err != nil {
			slog.Warn("cost-sync network failed", "network", credRow.Network, "customer_id", credRow.CustomerID.Bytes, "error", err)
			metrics.CostSyncRunsTotal.WithLabelValues("failed").Inc()
		}
	}
	return nil
}

func (w *Worker) syncCredential(ctx context.Context, credRow db.CostSyncCredential, date time.Time, trigger string) error {
	return w.syncCredentialRun(ctx, credRow, date, trigger, syncRunOpts{reconcile: true})
}

func (w *Worker) syncCredentialSubdaily(ctx context.Context, credRow db.CostSyncCredential, date time.Time, trigger string) error {
	opts := syncRunOpts{
		reconcile:    false,
		attribute:    true,
		snapshotHour: time.Now().UTC().Truncate(time.Hour),
		tokenMapping: ParseTokenMapping(credRow.TokenMapping),
	}
	err := w.syncCredentialRun(ctx, credRow, date, trigger, opts)
	w.bumpCredentialNextRun(ctx, credRow)
	return err
}

type syncRunOpts struct {
	reconcile    bool
	attribute    bool
	snapshotHour time.Time
	tokenMapping TokenMapping
}

func (w *Worker) syncCredentialRun(ctx context.Context, credRow db.CostSyncCredential, date time.Time, trigger string, opts syncRunOpts) error {
	start := time.Now()
	network := credRow.Network

	run, err := db.New(w.pool).InsertCostSyncRun(ctx, db.InsertCostSyncRunParams{
		CustomerID:    credRow.CustomerID,
		Network:       network,
		CostDate:      pgtype.Date{Time: date, Valid: true},
		Status:        "RUNNING",
		TriggerSource: trigger,
	})
	if err != nil {
		return err
	}

	cred, err := w.DecryptCredential(credRow)
	if err != nil {
		w.completeRun(ctx, run.ID, "FAILED", 0, 0, err.Error())
		return err
	}

	if err := w.MaybeRefreshToken(ctx, network, credRow, &cred); err != nil {
		w.completeRun(ctx, run.ID, "FAILED", 0, 0, err.Error())
		return err
	}

	lines, err := provider.FetchNetworkCosts(ctx, w.httpClient, network, w.networkBaseURL[network], cred, date)
	if err != nil {
		w.completeRun(ctx, run.ID, "FAILED", 0, 0, err.Error())
		metrics.CostSyncRunsTotal.WithLabelValues("failed").Inc()
		return err
	}
	if opts.attribute {
		ApplyNetworkObjectToken(lines, opts.tokenMapping)
	}
	if !opts.snapshotHour.IsZero() {
		for i := range lines {
			lines[i].SnapshotHour = opts.snapshotHour
		}
	}

	imported, totalUSD, usdAmounts, err := w.persistLines(ctx, lines, date)
	if err != nil {
		w.completeRun(ctx, run.ID, "FAILED", imported, totalUSD, err.Error())
		metrics.CostSyncRunsTotal.WithLabelValues("failed").Inc()
		return err
	}

	if opts.attribute && w.clickAttributor != nil {
		if err := w.clickAttributor.AttributeLines(ctx, run.ID, uuid.New(), opts.tokenMapping, lines, usdAmounts, date); err != nil {
			slog.Warn("cost-sync click attribution failed", "network", network, "error", err)
		}
	}

	if opts.reconcile {
		if err := w.reconcileCampaigns(ctx, lines, date); err != nil {
			slog.Warn("cost-sync reconciliation partial failure", "error", err)
		}
	}

	w.completeRun(ctx, run.ID, "COMPLETED", imported, totalUSD, "")
	metrics.CostSyncRunsTotal.WithLabelValues("success").Inc()
	metrics.CostSyncLastSuccessTimestamp.WithLabelValues(network).Set(float64(time.Now().Unix()))
	metrics.CostSyncRowsImported.Add(float64(imported))
	metrics.CostSyncDurationSeconds.WithLabelValues(network).Observe(time.Since(start).Seconds())

	if w.onSyncComplete != nil {
		w.onSyncComplete(network, time.Since(start))
	}
	return nil
}

func (w *Worker) persistLines(ctx context.Context, lines []CostLine, date time.Time) (int, int64, []int64, error) {
	if len(lines) == 0 {
		return 0, 0, nil, nil
	}

	fxCache, err := w.converter.PrepareFXCache(ctx, lines, date)
	if err != nil {
		return 0, 0, nil, err
	}

	usdAmounts := make([]int64, len(lines))
	var totalUSD int64
	for i, line := range lines {
		usdMicro, err := w.converter.ToUSDMicroCached(line.AmountMicro, line.Currency, fxCache)
		if err != nil {
			return 0, 0, nil, err
		}
		usdAmounts[i] = usdMicro
		totalUSD += usdMicro
	}

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return 0, 0, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	imported, err := insertCampaignCostsBatch(ctx, tx, lines, usdAmounts)
	if err != nil {
		return 0, 0, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return imported, totalUSD, usdAmounts, err
	}

	if w.insertSnapshots != nil {
		if err := w.insertSnapshots(ctx, lines, usdAmounts); err != nil {
			slog.Warn("cost-sync clickhouse insert failed", "error", err)
			metrics.CostSyncClickHouseErrors.Inc()
		}
	}

	return imported, totalUSD, usdAmounts, nil
}

func (w *Worker) bumpCredentialNextRun(ctx context.Context, row db.CostSyncCredential) {
	interval := int(row.SyncIntervalMinutes)
	if interval <= 0 {
		interval = 1440
	}
	if !ValidSyncIntervalMinutes(interval) {
		return
	}
	next := time.Now().UTC().Add(time.Duration(interval) * time.Minute)
	_ = db.New(w.pool).UpdateCostSyncCredentialNextRun(ctx, db.UpdateCostSyncCredentialNextRunParams{
		CustomerID: row.CustomerID,
		Network:    row.Network,
		NextRunAt:  pgtype.Timestamptz{Time: next, Valid: true},
	})
}
