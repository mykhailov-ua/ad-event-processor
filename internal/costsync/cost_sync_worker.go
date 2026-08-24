package costsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/postback"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	costSyncAdvisoryLockKey = int64(0x657370785f636f73)
	advisoryUnlockTimeout   = 5 * time.Second
)

type OAuthConfig struct {
	MetaAppID, MetaAppSecret           string
	GoogleClientID, GoogleClientSecret string
}

type Worker struct {
	pool            *pgxpool.Pool
	converter       *CurrencyConverter
	encryptionKey   []byte
	httpClient      *http.Client
	networkBaseURL  map[string]string
	oauth           OAuthConfig
	insertSnapshots func(context.Context, []CostLine, []int64) error
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
	slog.Info("cost-sync worker starting", "interval", "1h")
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	w.runHourlyGuarded(ctx, "cron")

	for {
		select {
		case <-ctx.Done():
			w.Wait()
			return
		case <-ticker.C:
			w.runHourlyGuarded(ctx, "cron")
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

	cred, err := w.decryptCredential(credRow)
	if err != nil {
		w.completeRun(ctx, run.ID, "FAILED", 0, 0, err.Error())
		return err
	}

	if err := w.maybeRefreshToken(ctx, network, credRow, &cred); err != nil {
		w.completeRun(ctx, run.ID, "FAILED", 0, 0, err.Error())
		return err
	}

	lines, err := fetchNetworkCosts(ctx, w.httpClient, network, w.networkBaseURL[network], cred, date)
	if err != nil {
		w.completeRun(ctx, run.ID, "FAILED", 0, 0, err.Error())
		metrics.CostSyncRunsTotal.WithLabelValues("failed").Inc()
		return err
	}

	imported, totalUSD, err := w.persistLines(ctx, lines, date)
	if err != nil {
		w.completeRun(ctx, run.ID, "FAILED", imported, totalUSD, err.Error())
		metrics.CostSyncRunsTotal.WithLabelValues("failed").Inc()
		return err
	}

	if err := w.reconcileCampaigns(ctx, lines, date); err != nil {
		slog.Warn("cost-sync reconciliation partial failure", "error", err)
	}

	w.completeRun(ctx, run.ID, "COMPLETED", imported, totalUSD, "")
	metrics.CostSyncRunsTotal.WithLabelValues("success").Inc()
	metrics.CostSyncRowsImported.Add(float64(imported))
	metrics.CostSyncDurationSeconds.WithLabelValues(network).Observe(time.Since(start).Seconds())

	if w.onSyncComplete != nil {
		w.onSyncComplete(network, time.Since(start))
	}
	return nil
}

func (w *Worker) persistLines(ctx context.Context, lines []CostLine, date time.Time) (int, int64, error) {
	if len(lines) == 0 {
		return 0, 0, nil
	}

	fxCache, err := w.converter.PrepareFXCache(ctx, lines, date)
	if err != nil {
		return 0, 0, err
	}

	usdAmounts := make([]int64, len(lines))
	var totalUSD int64
	for i, line := range lines {
		usdMicro, err := w.converter.ToUSDMicroCached(line.AmountMicro, line.Currency, fxCache)
		if err != nil {
			return 0, 0, err
		}
		usdAmounts[i] = usdMicro
		totalUSD += usdMicro
	}

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	imported, err := insertCampaignCostsBatch(ctx, tx, lines, usdAmounts)
	if err != nil {
		return 0, 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return imported, totalUSD, err
	}

	if w.insertSnapshots != nil {
		if err := w.insertSnapshots(ctx, lines, usdAmounts); err != nil {
			slog.Warn("cost-sync clickhouse insert failed", "error", err)
			metrics.CostSyncCHErrors.Inc()
		}
	}

	return imported, totalUSD, nil
}

func (w *Worker) reconcileCampaigns(ctx context.Context, lines []CostLine, date time.Time) error {
	seen := make(map[uuid.UUID]struct{})
	for _, line := range lines {
		if line.LineType != LineTypeSpend {
			continue
		}
		seen[line.CampaignID] = struct{}{}
	}
	if len(seen) == 0 {
		return nil
	}

	campaignIDs := make([]uuid.UUID, 0, len(seen))
	for campID := range seen {
		campaignIDs = append(campaignIDs, campID)
	}

	rows, err := w.pool.Query(ctx, `
		SELECT
			c.id,
			c.customer_id,
			COALESCE(cc.api_spend, 0) AS api_spend,
			COALESCE(tr.tracker_spend, 0) AS tracker_spend
		FROM campaigns c
		LEFT JOIN (
			SELECT campaign_id, SUM(amount_usd_micro)::bigint AS api_spend
			FROM campaign_costs
			WHERE cost_date = $1 AND line_type = 'spend'
			GROUP BY campaign_id
		) cc ON cc.campaign_id = c.id
		LEFT JOIN (
			SELECT campaign_id,
				COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0)::bigint AS tracker_spend
			FROM balance_ledger
			WHERE created_at >= $1::date
			  AND created_at < ($1::date + INTERVAL '1 day')
			  AND type IN ('FEE', 'RECONCILIATION_ADJUST', 'REFUND')
			GROUP BY campaign_id
		) tr ON tr.campaign_id = c.id
		WHERE c.id = ANY($2)
		  AND COALESCE(cc.api_spend, 0) != COALESCE(tr.tracker_spend, 0)`, date, campaignIDs)
	if err != nil {
		return err
	}
	defer rows.Close()

	batch := &pgx.Batch{}
	var adjustments int
	for rows.Next() {
		var campID, customerID uuid.UUID
		var apiSpend, trackerSpend int64
		if err := rows.Scan(&campID, &customerID, &apiSpend, &trackerSpend); err != nil {
			return err
		}
		delta := apiSpend - trackerSpend
		hash := reconciliationHash(customerID, campID, date)
		batch.Queue(`
			INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, idempotency_hash)
			VALUES ($1, $2, $3, 'RECONCILIATION_ADJUST', $4)
			ON CONFLICT (idempotency_hash) DO NOTHING`,
			pgtype.UUID{Bytes: customerID, Valid: true},
			pgtype.UUID{Bytes: campID, Valid: true},
			-delta,
			hash,
		)
		adjustments++
		metrics.CostSyncReconciliationDelta.Add(float64(abs64(delta)))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if batch.Len() == 0 {
		return nil
	}
	br := w.pool.SendBatch(ctx, batch)
	for i := 0; i < adjustments; i++ {
		if _, err := br.Exec(); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			_ = br.Close()
			return err
		}
	}
	return br.Close()
}

func reconciliationHash(customerID, campaignID uuid.UUID, date time.Time) string {
	raw := fmt.Sprintf("cost_sync_recon|%s|%s|%s", customerID, campaignID, date.Format("2006-01-02"))
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (w *Worker) decryptCredential(row db.CostSyncCredential) (Credential, error) {
	cred := Credential{
		CustomerID: row.CustomerID.Bytes,
		Network:    row.Network,
		AccountID:  row.AccountID,
	}
	if len(row.AccessTokenEncrypted) > 0 {
		b, err := postback.DecryptAESGCM(row.AccessTokenEncrypted, w.encryptionKey)
		if err != nil {
			return cred, err
		}
		cred.AccessToken = string(b)
	}
	if len(row.RefreshTokenEncrypted) > 0 {
		b, err := postback.DecryptAESGCM(row.RefreshTokenEncrypted, w.encryptionKey)
		if err != nil {
			return cred, err
		}
		cred.RefreshToken = string(b)
	}
	if len(row.ApiKeyEncrypted) > 0 {
		b, err := postback.DecryptAESGCM(row.ApiKeyEncrypted, w.encryptionKey)
		if err != nil {
			return cred, err
		}
		cred.APIKey = string(b)
	}
	if len(row.ExtraConfig) > 0 {
		if err := json.Unmarshal(row.ExtraConfig, &cred.ExtraConfig); err != nil {
			return cred, fmt.Errorf("parse extra_config for network=%s account=%s: %w", row.Network, row.AccountID, err)
		}
	}
	if row.TokenExpiresAt.Valid {
		cred.ExpiresAt = row.TokenExpiresAt.Time
	}
	return cred, nil
}

func (w *Worker) maybeRefreshToken(ctx context.Context, network string, row db.CostSyncCredential, cred *Credential) error {
	if !cred.ExpiresAt.IsZero() && time.Until(cred.ExpiresAt) > 5*time.Minute {
		return nil
	}

	var (
		token   string
		expires time.Time
		err     error
	)
	switch network {
	case "facebook":
		if w.oauth.MetaAppID == "" || w.oauth.MetaAppSecret == "" {
			return nil
		}
		token, expires, err = refreshMetaOAuth(ctx, w.httpClient, w.oauth.MetaAppID, w.oauth.MetaAppSecret, *cred)
	case "google":
		if w.oauth.GoogleClientID == "" || w.oauth.GoogleClientSecret == "" {
			return nil
		}
		token, expires, err = refreshGoogleOAuth(ctx, w.httpClient, w.oauth.GoogleClientID, w.oauth.GoogleClientSecret, *cred)
	default:
		return nil
	}
	if err != nil {
		return err
	}
	cred.AccessToken = token
	cred.ExpiresAt = expires

	enc, err := postback.EncryptAESGCM([]byte(token), w.encryptionKey)
	if err != nil {
		return err
	}
	_, err = db.New(w.pool).UpsertCostSyncCredential(ctx, db.UpsertCostSyncCredentialParams{
		CustomerID:            row.CustomerID,
		Network:               row.Network,
		AccountID:             row.AccountID,
		AccessTokenEncrypted:  enc,
		RefreshTokenEncrypted: row.RefreshTokenEncrypted,
		ApiKeyEncrypted:       row.ApiKeyEncrypted,
		ExtraConfig:           row.ExtraConfig,
		TokenExpiresAt:        pgtype.Timestamptz{Time: expires, Valid: true},
	})
	return err
}

func (w *Worker) completeRun(ctx context.Context, id int64, status string, rows int, totalUSD int64, errMsg string) {
	var msg pgtype.Text
	if errMsg != "" {
		msg = pgtype.Text{String: errMsg, Valid: true}
	}
	_ = db.New(w.pool).CompleteCostSyncRun(ctx, db.CompleteCostSyncRunParams{
		ID:                  id,
		Status:              status,
		RowsImported:        int32(rows),
		TotalAmountUsdMicro: totalUSD,
		ErrorMessage:        msg,
	})
}

func (w *Worker) tryAdvisoryLock(ctx context.Context) (bool, error) {
	var ok bool
	err := w.pool.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, costSyncAdvisoryLockKey).Scan(&ok)
	return ok, err
}

func (w *Worker) releaseAdvisoryLock(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, advisoryUnlockTimeout)
	defer cancel()
	_, err := w.pool.Exec(ctx, `SELECT pg_advisory_unlock($1)`, costSyncAdvisoryLockKey)
	if err == nil {
		return
	}
	if errors.Is(err, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded {
		slog.Error("cost-sync advisory unlock timed out", "error", err)
		return
	}
	slog.Warn("cost-sync advisory unlock failed", "error", err)
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func EncryptCredentialFields(key []byte, accessToken, refreshToken, apiKey string) (accessEnc, refreshEnc, apiEnc []byte, err error) {
	key = normalizeKey(key)
	if accessToken != "" {
		accessEnc, err = postback.EncryptAESGCM([]byte(accessToken), key)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if refreshToken != "" {
		refreshEnc, err = postback.EncryptAESGCM([]byte(refreshToken), key)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if apiKey != "" {
		apiEnc, err = postback.EncryptAESGCM([]byte(apiKey), key)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return accessEnc, refreshEnc, apiEnc, nil
}
