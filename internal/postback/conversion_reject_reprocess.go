package postback

import (
	"context"
	"log/slog"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
)

const (
	conversionReprocessCHTimeout = 30 * time.Second
	conversionReprocessBatchMax  = 500
)

// ConversionFraudTelemetryWriter inserts rejected conversions into ClickHouse fraud_events.
type ConversionFraudTelemetryWriter interface {
	WriteFraudTelemetry(ctx context.Context, events []*domain.Event) error
}

// ConversionRowWriter replaces pending conversion rows with validated inserts.
type ConversionRowWriter interface {
	ReplaceValidatedConversions(ctx context.Context, events []*domain.Event) error
}

// ConversionPayoutBatchApplier resolves payout on validated conversion batches (reprocess path).
type ConversionPayoutBatchApplier interface {
	ApplyBatch(ctx context.Context, events []*domain.Event)
}

// ConversionRejectReprocessor replays smart reject on conversions deferred during store outages.
type ConversionRejectReprocessor struct {
	cfg      config.ConversionReject
	ch       *database.CHQuery
	applier  *ConversionRejectApplier
	fraud    ConversionFraudTelemetryWriter
	rows     ConversionRowWriter
	payout   ConversionPayoutBatchApplier
	postback *ConversionPostbackEnqueuer
}

func NewConversionRejectReprocessor(
	cfg config.ConversionReject,
	ch *database.CHQuery,
	applier *ConversionRejectApplier,
	fraud ConversionFraudTelemetryWriter,
	rows ConversionRowWriter,
	payout ConversionPayoutBatchApplier,
	postback *ConversionPostbackEnqueuer,
) *ConversionRejectReprocessor {
	if applier == nil || ch == nil || !cfg.Enabled || !cfg.ReprocessEnabled {
		return nil
	}
	return &ConversionRejectReprocessor{
		cfg:      cfg,
		ch:       ch,
		applier:  applier,
		fraud:    fraud,
		rows:     rows,
		payout:   payout,
		postback: postback,
	}
}

func (r *ConversionRejectReprocessor) Start(ctx context.Context) {
	if r == nil {
		return
	}
	interval := time.Duration(r.cfg.ReprocessIntervalMin) * time.Minute
	if interval < 5*time.Minute {
		interval = 5 * time.Minute
	}
	if interval > 60*time.Minute {
		interval = 60 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	r.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.tick(ctx)
		}
	}
}

func (r *ConversionRejectReprocessor) tick(ctx context.Context) {
	if r == nil || r.ch == nil || r.applier == nil {
		return
	}
	lookback := time.Duration(r.cfg.ReprocessLookbackHours) * time.Hour
	if lookback < time.Hour {
		lookback = time.Hour
	}
	if lookback > 7*24*time.Hour {
		lookback = 7 * 24 * time.Hour
	}
	end := time.Now().UTC().Add(-2 * time.Minute)
	start := end.Add(-lookback)
	events, err := r.loadPendingConversions(ctx, start, end)
	if err != nil {
		slog.Warn("conversion smart reject reprocess load failed", "error", err)
		metrics.ConversionRejectReprocessTotal.WithLabelValues("load_error").Inc()
		return
	}
	if len(events) == 0 {
		return
	}
	for _, evt := range events {
		if evt == nil {
			continue
		}
		evt.Payload = clearPayloadBoolFlag(evt.Payload, domain.ConversionValidationPendingKey)
	}
	r.applier.ApplyBatch(ctx, events)

	rejected := make([]*domain.Event, 0, len(events))
	approved := make([]*domain.Event, 0, len(events))
	for _, evt := range events {
		if evt == nil {
			continue
		}
		if evt.FraudReason != "" {
			rejected = append(rejected, evt)
			continue
		}
		approved = append(approved, evt)
	}
	if len(rejected) > 0 && r.fraud != nil {
		if err := r.fraud.WriteFraudTelemetry(ctx, rejected); err != nil {
			slog.Warn("conversion smart reject reprocess fraud insert failed", "count", len(rejected), "error", err)
			metrics.ConversionRejectReprocessTotal.WithLabelValues("fraud_write_error").Inc()
		} else {
			metrics.ConversionRejectReprocessTotal.WithLabelValues("rejected").Add(float64(len(rejected)))
		}
	}
	if len(approved) > 0 {
		if r.payout != nil {
			r.payout.ApplyBatch(ctx, approved)
		}
		if r.rows != nil {
			if err := r.rows.ReplaceValidatedConversions(ctx, approved); err != nil {
				slog.Warn("conversion smart reject reprocess conversion update failed", "count", len(approved), "error", err)
				metrics.ConversionRejectReprocessTotal.WithLabelValues("row_write_error").Inc()
			} else {
				metrics.ConversionRejectReprocessTotal.WithLabelValues("approved").Add(float64(len(approved)))
				if r.postback != nil {
					r.postback.OnBatchStored(ctx, approved)
				}
			}
		}
	}
}

func (r *ConversionRejectReprocessor) loadPendingConversions(ctx context.Context, start, end time.Time) ([]*domain.Event, error) {
	chCtx, cancel := context.WithTimeout(ctx, conversionReprocessCHTimeout)
	defer cancel()
	rows, err := r.ch.Query(chCtx, `
SELECT click_id, campaign_id, payload, created_at, country
FROM conversions
WHERE created_at >= ?
 AND created_at < ?
 AND JSONExtractBool(payload, '`+domain.ConversionValidationPendingKey+`') = 1
LIMIT ?`, start, end, conversionReprocessBatchMax)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]*domain.Event, 0, conversionReprocessBatchMax)
	for rows.Next() {
		var clickID string
		var campaignID uuid.UUID
		var payload string
		var createdAt time.Time
		var country string
		if err := rows.Scan(&clickID, &campaignID, &payload, &createdAt, &country); err != nil {
			return nil, err
		}
		evt := &domain.Event{
			ClickID:    clickID,
			CampaignID: campaignID,
			Type:       "conversion",
			CreatedAt:  createdAt,
			GeoCountry: country,
			Payload:    []byte(payload),
		}
		mergeConversionPayloadFields(evt)
		out = append(out, evt)
	}
	return out, rows.Err()
}

func mergeConversionPayloadFields(evt *domain.Event) {
	if evt == nil {
		return
	}
	fields := parsePayloadStringFields(evt.Payload)
	if c := normalizeCountryCode(fields["country"]); c != "" && evt.GeoCountry == "" {
		evt.GeoCountry = c
	}
	if ip := fields["ip"]; ip != "" {
		evt.IP = ip
	}
}
