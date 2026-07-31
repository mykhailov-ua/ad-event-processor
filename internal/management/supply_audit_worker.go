package management

import (
	"context"
	"log/slog"
	"time"

	"espx/internal/database"
	db "espx/internal/ingestion/sqlc"
)

const supplyAuditInterval = 6 * time.Hour

type SupplyAuditWorker struct {
	svc      *Service
	interval time.Duration
}

func NewSupplyAuditWorker(svc *Service) *SupplyAuditWorker {
	return &SupplyAuditWorker{svc: svc, interval: supplyAuditInterval}
}

func (w *SupplyAuditWorker) Start(ctx context.Context) {
	if w == nil || w.svc == nil {
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

func (w *SupplyAuditWorker) tick(ctx context.Context) {
	report, err := w.svc.AuditSupplyCompliance(ctx)
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

type SupplyAuditReport struct {
	SellerCount int `json:"seller_count"`
	AdsTxtLines int `json:"ads_txt_lines"`
	Issues      int `json:"issues"`
}

func (s *Service) AuditSupplyCompliance(ctx context.Context) (SupplyAuditReport, error) {
	out := SupplyAuditReport{}
	if s == nil || s.pool == nil {
		return out, nil
	}
	q := db.New(s.pool)
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
	if _, err := s.BuildSellersJSON(ctx); err != nil {
		out.Issues++
	}
	if _, err := s.BuildAdsTxt(ctx); err != nil {
		out.Issues++
	}
	return out, nil
}
