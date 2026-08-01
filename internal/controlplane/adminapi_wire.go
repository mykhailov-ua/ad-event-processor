package controlplane

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"espx/internal/billing/plansyaml"
	"espx/internal/controlplane/adminapi"
	"espx/internal/controlplane/authz"
	"espx/internal/costsync"
	"espx/internal/domain"
	db "espx/internal/domain/db"
	"espx/pkg/supportbundle"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type campaignReaderAdapter struct{ svc *Service }

func (a campaignReaderAdapter) GetCampaign(ctx context.Context, campaignID uuid.UUID) (any, error) {
	return a.svc.GetCampaignDTO(ctx, campaignID)
}

func (a campaignReaderAdapter) GetCampaignMargin(ctx context.Context, campaignID uuid.UUID) (any, error) {
	if a.svc == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	return a.svc.GetCampaignMargin(ctx, campaignID)
}

type campaignStatsAdapter struct{ svc *Service }

func (a campaignStatsAdapter) GetCampaignStats(ctx context.Context, campaignID uuid.UUID, from, to time.Time, granularity string) (adminapi.CampaignStatsDTO, error) {
	report, err := a.svc.GetCampaignStats(ctx, campaignID, from, to, granularity)
	if err != nil {
		return adminapi.CampaignStatsDTO{}, err
	}
	hourly := make([]adminapi.CampaignHourlyBucketDTO, len(report.Hourly))
	for i, b := range report.Hourly {
		hourly[i] = adminapi.CampaignHourlyBucketDTO{
			Hour:        b.Hour,
			Impressions: b.Impressions,
			Clicks:      b.Clicks,
			Conversions: b.Conversions,
		}
	}
	return adminapi.CampaignStatsDTO{
		CampaignID:   report.CampaignID,
		CurrentSpend: report.CurrentSpend,
		Metrics: adminapi.CampaignMetricsDTO{
			Impressions: report.Metrics.Impressions,
			Clicks:      report.Metrics.Clicks,
			Conversions: report.Metrics.Conversions,
		},
		Hourly:      hourly,
		Granularity: report.Granularity,
		From:        report.From,
		To:          report.To,
		Stale:       report.Stale,
		Source:      report.Source,
		Consistency: report.Consistency,
	}, nil
}

type campaignForecastAdapter struct{ svc *Service }

func (a campaignForecastAdapter) ForecastCampaign(ctx context.Context, in adminapi.CampaignForecastInput) (adminapi.CampaignForecastDTO, error) {
	out, err := a.svc.ForecastCampaign(ctx, CampaignForecastInput{
		CustomerID:       in.CustomerID,
		BudgetLimitMicro: in.BudgetLimitMicro,
		TargetCountries:  in.TargetCountries,
		DaypartHours:     in.DaypartHours,
		StartAt:          in.StartAt,
		EndAt:            in.EndAt,
		PacingMode:       in.PacingMode,
		Timezone:         in.Timezone,
	})
	if err != nil {
		return adminapi.CampaignForecastDTO{}, err
	}
	curve := make([]adminapi.SpendCurvePoint, len(out.SpendCurve))
	for i, p := range out.SpendCurve {
		curve[i] = adminapi.SpendCurvePoint{
			Hour:        p.Hour,
			SpendMicro:  p.SpendMicro,
			Impressions: p.Impressions,
		}
	}
	var advisory *adminapi.ForecastAdvisory
	if out.Advisory != nil {
		advisory = &adminapi.ForecastAdvisory{
			Code:            out.Advisory.Code,
			Message:         out.Advisory.Message,
			SuggestedPacing: out.Advisory.SuggestedPacing,
		}
	}
	return adminapi.CampaignForecastDTO{
		ImpressionsP50: out.ImpressionsP50,
		ImpressionsP90: out.ImpressionsP90,
		SpendCurve:     curve,
		LowConfidence:  out.LowConfidence,
		Advisory:       advisory,
	}, nil
}

type customerBalanceAdapter struct{ svc *Service }

func (a customerBalanceAdapter) GetCustomerBalance(ctx context.Context, customerID uuid.UUID) (adminapi.CustomerBalanceDTO, error) {
	bal, err := a.svc.GetCustomerBalance(ctx, customerID)
	if err != nil {
		return adminapi.CustomerBalanceDTO{}, err
	}
	ledger := make([]adminapi.BalanceLedgerDTO, len(bal.Ledger))
	for i, row := range bal.Ledger {
		ledger[i] = adminapi.BalanceLedgerDTO{
			ID:              row.ID,
			CustomerID:      row.CustomerID,
			CampaignID:      row.CampaignID,
			Amount:          row.Amount,
			Type:            row.Type,
			IdempotencyHash: row.IdempotencyHash,
			CreatedAt:       row.CreatedAt,
		}
	}
	return adminapi.CustomerBalanceDTO{
		CustomerID: bal.CustomerID,
		Balance:    bal.Balance,
		Currency:   bal.Currency,
		Ledger:     ledger,
	}, nil
}

func (a customerBalanceAdapter) ExportCustomerLedgerCSV(ctx context.Context, customerID uuid.UUID, cursor int64, w io.Writer) (adminapi.LedgerExportResult, error) {
	result, err := a.svc.ExportCustomerLedgerCSV(ctx, customerID, cursor, w)
	if err != nil {
		return adminapi.LedgerExportResult{}, err
	}
	return adminapi.LedgerExportResult{
		NextCursor: result.NextCursor,
		Truncated:  result.Truncated,
		Bytes:      result.Bytes,
	}, nil
}

type disputeListerAdapter struct {
	payment *PaymentClient
	pool    *Service
}

func (a disputeListerAdapter) ListDisputes(ctx context.Context, customerFilter string, limit, offset int32) (adminapi.DisputeListResult, error) {
	if a.payment == nil {
		return adminapi.DisputeListResult{}, status.Error(codes.Unavailable, "payment service not configured")
	}
	resp, err := a.payment.ListDisputes(ctx, customerFilter, limit, offset)
	if err != nil {
		return adminapi.DisputeListResult{}, err
	}
	queries := db.New(a.pool.GetPool())
	rows := make([]adminapi.DisputeRowDTO, 0, len(resp.Disputes))
	for _, d := range resp.Disputes {
		item := adminapi.DisputeRowDTO{
			IntentID:          d.IntentID,
			CustomerID:        d.CustomerID,
			AmountMicro:       d.AmountMicro,
			Currency:          d.Currency,
			ProviderDisputeID: d.ProviderDisputeID,
		}
		if !d.UpdatedAt.IsZero() {
			item.UpdatedAt = d.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		}
		intentID, parseErr := uuid.Parse(d.IntentID)
		if parseErr == nil {
			ledgerIDs, lerr := queries.ListLedgerChargebackEntryIDs(ctx, pgtype.UUID{Bytes: intentID, Valid: true})
			if lerr == nil && len(ledgerIDs) > 0 {
				item.ChargebackLedgerEntryIDs = ledgerIDs
			} else {
				item.ChargebackLedgerEntryIDs = []int64{}
			}
		}
		rows = append(rows, item)
	}
	return adminapi.DisputeListResult{Disputes: rows, Total: resp.Total}, nil
}

type campaignAdminAdapter struct{ svc *Service }

func (a campaignAdminAdapter) EnforceSelfServeCreateLimits(ctx context.Context, customerID uuid.UUID, budgetMicro int64) error {
	return a.svc.EnforceSelfServeCreateLimits(ctx, customerID, budgetMicro)
}

func (a campaignAdminAdapter) GenerateIdempotencyHash(customerID uuid.UUID, payload []byte) (string, error) {
	return a.svc.GenerateIdempotencyHash(customerID, payload)
}

func (a campaignAdminAdapter) CreateCampaign(ctx context.Context, spec adminapi.CreateCampaignInput) (uuid.UUID, error) {
	pacing := db.PacingModeTypeASAP
	if spec.PacingMode == "EVEN" {
		pacing = db.PacingModeTypeEVEN
	}
	return a.svc.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID:      spec.CustomerID,
		BrandID:         spec.BrandID,
		Name:            spec.Name,
		BudgetLimit:     spec.BudgetLimitMicro,
		PacingMode:      pacing,
		DailyBudget:     spec.DailyBudgetMicro,
		Timezone:        spec.Timezone,
		FreqLimit:       spec.FreqLimit,
		FreqWindow:      spec.FreqWindow,
		TargetCountries: spec.TargetCountries,
		StartAt:         spec.StartAt,
		EndAt:           spec.EndAt,
		DaypartHours:    spec.DaypartHours,
		IdempotencyKey:  spec.IdempotencyKey,
	})
}

func (a campaignAdminAdapter) PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	return a.svc.PauseCampaign(ctx, campaignID, reason)
}

func (a campaignAdminAdapter) ResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	return a.svc.ResumeCampaign(ctx, campaignID, reason)
}

type apiKeyCreatorAdapter struct{ auth *AuthClient }

func (a apiKeyCreatorAdapter) CreateAPIKey(ctx context.Context, accessToken, name string) (adminapi.APIKeyResult, error) {
	if a.auth == nil {
		return adminapi.APIKeyResult{}, errAuthUnavailable
	}
	resp, err := a.auth.CreateAPIKey(ctx, accessToken, name)
	if err != nil {
		return adminapi.APIKeyResult{}, err
	}
	out := adminapi.APIKeyResult{
		ID:     resp.ID,
		Name:   resp.Name,
		RawKey: resp.RawKey,
	}
	if resp.ExpiresAt != nil {
		out.ExpiresAt = resp.ExpiresAt.UTC().Format(time.RFC3339)
		out.HasExpires = true
	}
	return out, nil
}

type consentRecorderAdapter struct{ svc *Service }

func (a consentRecorderAdapter) RecordConsent(ctx context.Context, in adminapi.ConsentRecord) error {
	return a.svc.RecordConsent(ctx, ConsentRecordInput{
		UserID:    in.UserID,
		Purposes:  in.Purposes,
		Source:    in.Source,
		Timestamp: in.Timestamp,
	})
}

type consentVerifierAdapter struct{ secret []byte }

func (v consentVerifierAdapter) Verify(body []byte, signature string) error {
	return VerifyConsentHMAC(v.secret, body, signature)
}

type blacklistAdapter struct{ svc *Service }

func (a blacklistAdapter) BlockIPWithTTL(ctx context.Context, ip, source string, ttlSeconds *int64) error {
	return a.svc.BlockIPWithTTL(ctx, ip, source, ttlSeconds)
}

func (a blacklistAdapter) PreviewBlockIP(ctx context.Context, ip, source string, ttlSeconds *int64) (any, error) {
	return a.svc.PreviewBlockIP(ctx, ip, source, ttlSeconds)
}

func (a blacklistAdapter) UnblockIP(ctx context.Context, ip, source string) error {
	return a.svc.UnblockIP(ctx, ip, source)
}

func (a blacklistAdapter) ListBlacklist(ctx context.Context, limit, offset int32) (any, int64, error) {
	return a.svc.ListBlacklist(ctx, limit, offset)
}

type fraudThreatAdapter struct{ svc *Service }

func (a fraudThreatAdapter) EnqueueFraudThreat(ctx context.Context, action, ip, campaignID string, score float64, boost int32, ttlSeconds int64) error {
	if a.svc == nil {
		return fmt.Errorf("service unavailable")
	}
	return a.svc.EnqueueFraudThreat(ctx, FraudThreatPayload{
		Action:     action,
		IP:         ip,
		CampaignID: campaignID,
		Score:      score,
		Boost:      boost,
		TTLSeconds: ttlSeconds,
	})
}

type supportBundleWriter struct {
	pool   *pgxpool.Pool
	logDir string
}

func (w supportBundleWriter) WriteSupportBundle(ctx context.Context, out io.Writer) error {
	meta := supportbundle.Meta{}
	if w.pool != nil {
		var dep uuid.UUID
		var state string
		err := w.pool.QueryRow(ctx, `
			SELECT deployment_id, state
			FROM billing.license_status
			LIMIT 1`).Scan(&dep, &state)
		if err != nil && err != pgx.ErrNoRows {
			return err
		}
		if err == nil {
			if dep != uuid.Nil {
				meta.DeploymentID = dep.String()
			}
			meta.LicenseState = state
		}
	}
	return supportbundle.Write(ctx, out, supportbundle.Options{
		Meta:     meta,
		LogDir:   w.logDir,
		MaxBytes: supportbundle.DefaultMaxBytes,
	})
}

type supportFeedbackAdapter struct {
	svc *Service
}

func (a supportFeedbackAdapter) SupportFeedbackMeta(ctx context.Context) (adminapi.SupportFeedbackMeta, error) {
	meta, err := a.svc.SupportFeedbackMeta(ctx)
	if err != nil {
		return adminapi.SupportFeedbackMeta{}, err
	}
	return adminapi.SupportFeedbackMeta{
		DeploymentID:  meta.DeploymentID,
		BinaryVersion: meta.BinaryVersion,
		SKU:           meta.SKU,
	}, nil
}

func (a supportFeedbackAdapter) RecordSupportFeedback(ctx context.Context, in adminapi.SupportFeedbackRecord) (uuid.UUID, error) {
	return a.svc.RecordSupportFeedback(ctx, SupportFeedbackInput{
		Type:          in.Type,
		ContactEmail:  in.ContactEmail,
		Message:       in.Message,
		AttachBundle:  in.AttachBundle,
		BundleGzip:    in.BundleGzip,
		SubmitterID:   in.SubmitterID,
		DeploymentID:  in.DeploymentID,
		BinaryVersion: in.BinaryVersion,
		SKU:           in.SKU,
	})
}

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
		Shards:           mapShardStatuses(report.Shards),
		Outbox: adminapi.OutboxHealthSummary{
			Pending:              report.Outbox.Pending,
			OldestPendingSeconds: report.Outbox.OldestPendingSeconds,
			LastProcessedEventID: report.Outbox.LastProcessedEventID,
		},
		StreamLag:     []adminapi.ShardStreamLag{},
		BreakerStates: map[string]string{},
	}, nil
}

func mapShardStatuses(in []ShardHealthStatus) []adminapi.ShardHealthStatus {
	out := make([]adminapi.ShardHealthStatus, len(in))
	for i, s := range in {
		out[i] = adminapi.ShardHealthStatus{
			ShardID:             s.ShardID,
			PingOK:              s.PingOK,
			PingError:           s.PingError,
			PingLatencyMs:       s.PingLatencyMs,
			ConfigVersion:       s.ConfigVersion,
			ConfigVersionLag:    s.ConfigVersionLag,
			ConfigVersionSynced: s.ConfigVersionSynced,
		}
	}
	return out
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
	_ = ctx
	_ = cursor
	_ = limit
	return adminapi.FanOutResult[adminapi.DLQEntryDTO]{Items: []adminapi.DLQEntryDTO{}}, nil
}

func (r *opsReader) EnqueueDLQRetry(ctx context.Context, payload adminapi.DLQRetryPayload, idempotencyKey string) error {
	_ = ctx
	_ = payload
	_ = idempotencyKey
	return fmt.Errorf("dlq retry not configured")
}

func (r *opsReader) GetShardHealthFanOut(ctx context.Context) (adminapi.ShardHealthAPIResponse, error) {
	report, err := r.svc.GetShardHealth(ctx)
	if err != nil {
		return adminapi.ShardHealthAPIResponse{}, err
	}
	return adminapi.ShardHealthAPIResponse{
		ShardHealthReport: adminapi.ShardHealthReport{
			EmergencyBreaker: report.EmergencyBreaker,
			Outbox: adminapi.OutboxHealthSummary{
				Pending:              report.Outbox.Pending,
				OldestPendingSeconds: report.Outbox.OldestPendingSeconds,
				LastProcessedEventID: report.Outbox.LastProcessedEventID,
			},
			Shards: mapShardStatuses(report.Shards),
		},
	}, nil
}

func (r *opsReader) ExportAuditCSV(ctx context.Context, cursor string, w io.Writer) (adminapi.AuditExportResult, error) {
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
	runs, total, err := r.svc.ListReconRuns(ctx, service, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]adminapi.ReconRunDTO, len(runs))
	for i, run := range runs {
		out[i] = adminapi.ReconRunDTO{
			Service:            run.Service,
			ID:                 run.ID,
			PeriodStart:        run.PeriodStart,
			PeriodEnd:          run.PeriodEnd,
			Status:             run.Status,
			TotalDelta:         run.TotalDelta,
			CampaignsChecked:   run.CampaignsChecked,
			DiscrepanciesFound: run.DiscrepanciesFound,
			FindingsCount:      run.FindingsCount,
			IntentsChecked:     run.IntentsChecked,
			ErrorMessage:       run.ErrorMessage,
			CreatedAt:          run.CreatedAt,
			CompletedAt:        run.CompletedAt,
		}
	}
	return out, total, nil
}

type auditLister struct{ svc *Service }

func (a auditLister) ListAuditLogs(ctx context.Context, limit, offset int32, redactPII bool) (any, int64, error) {
	return a.svc.ListAuditLogsRedacted(ctx, limit, offset, redactPII)
}

type rolesReloader struct{ mw *AuthMiddleware }

func (r rolesReloader) ReloadRoles() error {
	if r.mw == nil || r.mw.policy == nil {
		return fmt.Errorf("policy store not configured")
	}
	return authz.LoadRolesYAML(authz.DefaultRolesPath(), r.mw.policy)
}

func (r rolesReloader) RolesPath() string { return authz.DefaultRolesPath() }

type plansReloader struct {
	pool *pgxpool.Pool
	svc  *Service
}

func (p plansReloader) ReloadPlans(ctx context.Context, dryRun bool) (plansyaml.ReloadReport, error) {
	return plansyaml.Reload(ctx, p.pool, plansyaml.DefaultPlansPath(), dryRun, func(ctx context.Context) error {
		if dryRun || p.svc == nil {
			return nil
		}
		return p.svc.publishRegistryFullSync(ctx)
	})
}

func (p plansReloader) PlansPath() string { return plansyaml.DefaultPlansPath() }

func (h *Handler) BuildAdminAPIRegistry(pool *pgxpool.Pool, rdbs []redis.UniversalClient) adminapi.RouteRegistry {
	if h == nil || h.svc == nil || pool == nil {
		return adminapi.RouteRegistry{}
	}

	limit := h.limit
	perm := h.adminRequirePermission()
	permAny := h.adminRequireAnyPermission()
	selfServePerm := h.adminSelfServePermission()
	writeErr := func(w http.ResponseWriter, err error) {
		writeServiceError(w, err)
	}
	authCustomer := h.authorizeCustomerAccess
	authCampaign := h.authorizeCampaignAccess
	isAdmin := func(r *http.Request) bool {
		u, ok := GetUser(r.Context())
		return ok && !u.IsUser()
	}

	svc := h.svc
	composite := adminapi.NewCompositeReadService(pool, h.cfg, nil)
	if composite != nil {
		composite.SetCHQuery(svc.CHQuery())
	}

	exportDir := os.Getenv("BILLING_EXPORT_PATH")
	if exportDir == "" {
		exportDir = filepath.Join(".", "data", "billing-exports")
	}
	var exportHTTP *adminapi.ExportHTTPHandlers
	if composite != nil {
		exportHTTP = &adminapi.ExportHTTPHandlers{
			JobRunner:               adminapi.NewJobRunner(composite, exportDir),
			ApplyRateLimit:          limit,
			RequirePermission:       perm,
			AuthorizeCustomerAccess: authCustomer,
			WriteServiceError:       writeErr,
		}
	}

	encKey := []byte(h.cfg.ConsentHMACSecret)
	if len(encKey) < 32 && len(h.cfg.TokenSymmetricKey) >= 32 {
		encKey = []byte(h.cfg.TokenSymmetricKey)
	}

	var costWorker *costsync.Worker
	if h.cfg != nil && h.cfg.Control.EnableCostSync && len(encKey) >= 32 {
		costWorker = costsync.NewWorker(pool, encKey)
	}

	opsReader := newOpsReader(svc)

	return adminapi.RouteRegistry{
		BillingHTTP: &adminapi.BillingHTTPHandlers{
			Billing:                      h.billing,
			CompositeReads:               composite,
			ApplyRateLimit:               limit,
			RequirePermission:            perm,
			AuthorizeCustomerAccess:      authCustomer,
			WriteServiceError:            writeErr,
			RequestIsFromAdmin:           isAdmin,
			ApplySelfServeRateLimit:      limit,
			RequireSelfServePermission:   selfServePerm,
			ResolveSelfServeCustomerID:   h.resolveSelfServeCustomerIDForBilling,
			CustomerBalance:              customerBalanceAdapter{svc: svc},
			Disputes:                     disputeListerAdapter{payment: h.payment, pool: svc},
			LimitExportByCustomer:        h.limitExportByCustomer,
			ResolveDisputeCustomerFilter: h.resolveDisputeCustomerFilter,
		},
		OpsHTTP: &adminapi.OpsHTTPHandlers{
			OpsReader:               opsReader,
			PaymentIntents:          h.payment,
			ConsentRecorder:         consentRecorderAdapter{svc: svc},
			ConsentVerifier:         consentVerifierAdapter{secret: []byte(h.cfg.ConsentHMACSecret)},
			AuditLister:             auditLister{svc: svc},
			RolesReloader:           rolesReloader{mw: h.authMiddleware},
			PlansReloader:           plansReloader{pool: pool, svc: svc},
			Blacklist:               blacklistAdapter{svc: svc},
			FraudThreat:             fraudThreatAdapter{svc: svc},
			ApplyRateLimit:          limit,
			RequirePermission:       perm,
			WriteServiceError:       writeErr,
			AuthorizeCustomerAccess: authCustomer,
			SupportBundle: supportBundleWriter{
				pool:   pool,
				logDir: h.cfg.Logger.Dir,
			},
		},
		ExportHTTP: exportHTTP,
		LicensingHTTP: &adminapi.LicensingHTTPHandlers{
			Pool:                       pool,
			RedisForCustomer:           func(id uuid.UUID) redis.UniversalClient { return svc.getRDB(id) },
			ApplyRateLimit:             limit,
			RequirePermission:          perm,
			ApplySelfServeRateLimit:    limit,
			RequireSelfServePermission: selfServePerm,
			ResolveSelfServeCustomerID: h.resolveSelfServeCustomerIDForBilling,
		},
		ReportsHTTP: &adminapi.ReportsHTTPHandlers{
			CampaignStats:             campaignStatsAdapter{svc: svc},
			CampaignForecaster:        campaignForecastAdapter{svc: svc},
			Pool:                      pool,
			CHQuery:                   svc.CHQuery(),
			ApplyRateLimit:            limit,
			RequirePermission:         perm,
			RequireAnyPermission:      permAny,
			AuthorizeCampaignAccess:   authCampaign,
			ResolveForecastCustomerID: h.resolveForecastCustomerID,
			WriteServiceError:         writeErr,
		},
		DashboardsHTTP: &adminapi.DashboardsHTTPHandlers{
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		ViewsHTTP: &adminapi.ViewsHTTPHandlers{
			Service: adminapi.NewService(),
		},
		SelfServeHTTP: &adminapi.SelfServeHTTPHandlers{
			Campaigns:                  campaignAdminAdapter{svc: svc},
			PaymentIntents:             h.payment,
			Invoices:                   h.billing,
			APIKeys:                    apiKeyCreatorAdapter{auth: h.authClient},
			ApplyRateLimit:             limit,
			RequireSelfServePermission: selfServePerm,
			ResolveSelfServeCustomerID: h.resolveSelfServeCustomerIDForSelfServe,
			AuthorizeCampaignAccess:    authCampaign,
			WriteServiceError:          writeErr,
		},
		PostbackHTTP: &adminapi.PostbackHTTPHandlers{
			Pool:              pool,
			EncryptionKey:     encKey,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		CostSyncHTTP: &adminapi.CostSyncHTTPHandlers{
			Pool:              pool,
			EncryptionKey:     encKey,
			Worker:            costWorker,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		MarginGuardHTTP: &adminapi.MarginGuardHTTPHandlers{
			Service:           svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
		RtbFloorsHTTP: &adminapi.RtbFloorsHTTPHandlers{
			Service:           svc,
			ApplyRateLimit:    limit,
			RequirePermission: perm,
			WriteServiceError: writeErr,
		},
		CampaignsHTTP: &adminapi.CampaignsHTTPHandlers{
			Campaigns:               campaignReaderAdapter{svc: svc},
			ApplyRateLimit:          limit,
			RequireAnyPermission:    permAny,
			AuthorizeCampaignAccess: authCampaign,
			WriteServiceError:       writeErr,
		},
		SupportHTTP: &adminapi.SupportHTTPHandlers{
			Feedback: supportFeedbackAdapter{svc: svc},
			SupportBundle: supportBundleWriter{
				pool:   pool,
				logDir: h.cfg.Logger.Dir,
			},
			ApplyRateLimit:    limit,
			RequireAuth:       h.adminRequireAuth(),
			WriteServiceError: writeErr,
		},
		MetaHTTP: &adminapi.MetaHTTPHandlers{
			ApplyRateLimit: limit,
			Enrich:         h.metaEnricher(),
			WriteError:     writeErr,
		},
		StubHTTP: &adminapi.StubHTTPHandlers{
			ApplyRateLimit:    limit,
			RequirePermission: perm,
		},
	}
}

func (h *Handler) adminRequirePermission() func(string, http.HandlerFunc) http.HandlerFunc {
	return func(permission string, next http.HandlerFunc) http.HandlerFunc {
		return h.perm(next, permission)
	}
}

func (h *Handler) adminRequireAuth() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		if h.authMiddleware != nil {
			return h.authMiddleware.RequireAuth(RoleAdmin, RoleManager, RoleUser, RoleBuyer, RoleSupport)(next)
		}
		return h.authFallback(next)
	}
}

func (h *Handler) adminRequireAnyPermission() func([]string, http.HandlerFunc) http.HandlerFunc {
	return func(permissions []string, next http.HandlerFunc) http.HandlerFunc {
		if h.authMiddleware != nil {
			return h.authMiddleware.RequireAnyPermission(permissions...)(next)
		}
		return h.authFallback(next)
	}
}

func (h *Handler) adminSelfServePermission() func(string, http.HandlerFunc) http.HandlerFunc {
	return func(permission string, next http.HandlerFunc) http.HandlerFunc {
		return h.selfServePerm(next, permission)
	}
}

func (h *Handler) authorizeCustomerAccess(r *http.Request, customerID string) error {
	return h.ensureCustomerAccess(r, customerID)
}

func (h *Handler) authorizeCampaignAccess(r *http.Request, campaignID uuid.UUID) error {
	return h.ensureCampaignAccess(r, campaignID)
}

func (h *Handler) resolveSelfServeCustomerIDForBilling(r *http.Request) (uuid.UUID, error) {
	return h.resolveSelfServeCustomerID(r, nil)
}

func (h *Handler) resolveSelfServeCustomerIDForSelfServe(r *http.Request, bodyCustomerID *uuid.UUID) (uuid.UUID, error) {
	return h.resolveSelfServeCustomerID(r, bodyCustomerID)
}

func (h *Handler) resolveDisputeCustomerFilter(r *http.Request) (string, error) {
	u, ok := GetUser(r.Context())
	if !ok {
		return "", errForbidden
	}
	customerFilter := r.URL.Query().Get("customer_id")
	if u.IsUser() {
		if customerFilter != "" && customerFilter != u.CustomerID.String() {
			return "", errForbidden
		}
		return u.CustomerID.String(), nil
	}
	return customerFilter, nil
}
