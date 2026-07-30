package management

import (
	"context"
	"io"
	"time"

	"espx/internal/adminapi"
	db "espx/internal/ingestion/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type campaignReaderAdapter struct{ svc *Service }

func (a campaignReaderAdapter) GetCampaign(ctx context.Context, campaignID uuid.UUID) (any, error) {
	return a.svc.GetCampaignDTO(ctx, campaignID)
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
			IntentID:          d.IntentId,
			CustomerID:        d.CustomerId,
			AmountMicro:       d.AmountMicro,
			Currency:          d.Currency,
			ProviderDisputeID: d.ProviderDisputeId,
		}
		if d.UpdatedAt != nil {
			item.UpdatedAt = d.UpdatedAt.AsTime().UTC().Format("2006-01-02T15:04:05Z07:00")
		}
		intentID, parseErr := uuid.Parse(d.IntentId)
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

type paymentIntentsAdapter struct{ payment *PaymentClient }

func (a paymentIntentsAdapter) CreatePaymentIntent(ctx context.Context, customerID string, amountMicro int64, currency, idempotencyKey string, meta map[string]string) (adminapi.PaymentIntentResult, error) {
	if a.payment == nil {
		return adminapi.PaymentIntentResult{}, status.Error(codes.Unavailable, "payment service not configured")
	}
	resp, err := a.payment.CreatePaymentIntent(ctx, customerID, amountMicro, currency, idempotencyKey, meta)
	if err != nil {
		return adminapi.PaymentIntentResult{}, err
	}
	return adminapi.PaymentIntentResult{
		IntentID:    resp.IntentId,
		Status:      resp.Status.String(),
		CheckoutURL: resp.CheckoutUrl,
		ProviderRef: resp.ProviderRef,
	}, nil
}

func (a paymentIntentsAdapter) ListPaymentIntents(ctx context.Context, customerID string, limit, offset int32) (adminapi.PaymentIntentList, error) {
	if a.payment == nil {
		return adminapi.PaymentIntentList{}, status.Error(codes.Unavailable, "payment service not configured")
	}
	resp, err := a.payment.ListPaymentIntents(ctx, customerID, limit, offset)
	if err != nil {
		return adminapi.PaymentIntentList{}, err
	}
	items := make([]adminapi.PaymentIntentRow, 0, len(resp.Intents))
	for _, in := range resp.Intents {
		row := adminapi.PaymentIntentRow{
			ID:             in.Id,
			CustomerID:     in.CustomerId,
			AmountMicro:    in.AmountMicro,
			Currency:       in.Currency,
			Status:         in.Status.String(),
			Provider:       in.Provider,
			ProviderRef:    in.ProviderRef,
			IdempotencyKey: in.IdempotencyKey,
		}
		if in.CreatedAt != nil {
			row.CreatedAt = in.CreatedAt.AsTime().UTC().Format(time.RFC3339)
		}
		if in.UpdatedAt != nil {
			row.UpdatedAt = in.UpdatedAt.AsTime().UTC().Format(time.RFC3339)
		}
		items = append(items, row)
	}
	return adminapi.PaymentIntentList{Intents: items, Total: resp.Total}, nil
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
		ID:     resp.Id,
		Name:   resp.Name,
		RawKey: resp.RawKey,
	}
	if resp.ExpiresAt != nil {
		out.ExpiresAt = resp.ExpiresAt.AsTime().UTC().Format(time.RFC3339)
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
