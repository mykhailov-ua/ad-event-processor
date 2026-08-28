package supply

import (
	"context"
	"log/slog"
	"time"

	"ad-event-processor/internal/database"

	db "ad-event-processor/internal/domain/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

const auditInterval = 6 * time.Hour

type AuditHost interface {
	Pool() *pgxpool.Pool
	BuildSellersJSON(ctx context.Context) ([]byte, error)
	BuildAdsTxt(ctx context.Context) (string, error)
}

type AuditReport struct {
	SellerCount int `json:"seller_count"`
	AdsTxtLines int `json:"ads_txt_lines"`
	Issues      int `json:"issues"`
}

func AuditCompliance(ctx context.Context, host AuditHost) (AuditReport, error) {
	out := AuditReport{}
	if host == nil || host.Pool() == nil {
		return out, nil
	}
	q := db.New(host.Pool())
	sellers, err := q.ListSellers(ctx)
	if err != nil {
		return out, err
	}
	out.SellerCount = len(sellers)
	for _, row := range sellers {
		if row.Domain == "" || row.SellerID == "" {
			out.Issues++
		}
	}
	adsRows, err := q.ListAdsTxtEntries(ctx)
	if err != nil {
		return out, err
	}
	out.AdsTxtLines = len(adsRows)
	for _, row := range adsRows {
		if row.Domain == "" || row.PublisherAccountID == "" {
			out.Issues++
		}
	}
	if _, err := host.BuildSellersJSON(ctx); err != nil {
		out.Issues++
	}
	if _, err := host.BuildAdsTxt(ctx); err != nil {
		out.Issues++
	}
	return out, nil
}

type AuditWorker struct {
	host     AuditHost
	interval time.Duration
}

func NewAuditWorker(host AuditHost) *AuditWorker {
	return &AuditWorker{host: host, interval: auditInterval}
}

func NewSupplyAuditWorker(host AuditHost) *AuditWorker {
	return NewAuditWorker(host)
}

func (w *AuditWorker) Start(ctx context.Context) {
	if w == nil || w.host == nil {
		return
	}
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

func (w *AuditWorker) tick(ctx context.Context) {
	report, err := AuditCompliance(ctx, w.host)
	if err != nil {
		if database.IsShutdownError(err) {
			return
		}
		slog.Error("supply audit failed", "err", err)
		return
	}
	if report.Issues > 0 {
		slog.Warn("supply audit found issues",
			"issues", report.Issues,
			"sellers", report.SellerCount,
			"ads_txt_lines", report.AdsTxtLines,
		)
	}
}
