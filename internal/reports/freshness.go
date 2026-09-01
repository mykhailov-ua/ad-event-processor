package reports

import (
	"context"
	"encoding/csv"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
)

func finalizeDataFreshness(dto DataFreshnessDTO) DataFreshnessDTO {
	dto.AsOfDisplay = coldpath.RFC3339Display(dto.AsOf)
	return dto
}

func DataFreshnessFromClickHouse(ctx context.Context, clickhouseQuery *database.ClickHouseQuery) DataFreshnessDTO {
	dto := DataFreshnessDTO{
		AsOf:        time.Now().UTC().Format(time.RFC3339),
		Consistency: "eventual",
	}
	if clickhouseQuery == nil {
		dto.Stale = true
		return finalizeDataFreshness(dto)
	}
	lag, err := clickhouseQuery.IngestionLag(ctx)
	if err != nil {
		dto.Stale = true
		return finalizeDataFreshness(dto)
	}
	dto.Stale, dto.CHLagSeconds = database.Freshness(lag, 5*time.Minute)
	return finalizeDataFreshness(dto)
}

type DataSourceFreshnessDTO struct {
	Name         string `json:"name"`
	Consistency  string `json:"consistency"`
	Stale        bool   `json:"stale"`
	CHLagSeconds int    `json:"ch_lag_seconds,omitempty"`
}

func PortfolioFreshness(now time.Time, chQueryAvailable bool, chLag time.Duration) DataFreshnessDTO {
	chStale, lagSec := database.Freshness(chLag, 5*time.Minute)
	sources := []DataSourceFreshnessDTO{
		{Name: "counts", Consistency: "strong", Stale: false},
	}
	dto := DataFreshnessDTO{
		AsOf:        now.Format(time.RFC3339),
		Consistency: "strong",
		Stale:       !chQueryAvailable,
		Sources:     sources,
	}
	if !chQueryAvailable {
		return finalizeDataFreshness(dto)
	}
	sources = append(sources, DataSourceFreshnessDTO{
		Name:         "money",
		Consistency:  "eventual",
		Stale:        chStale,
		CHLagSeconds: lagSec,
	})
	dto.Sources = sources
	dto.Consistency = "mixed"
	dto.Stale = chStale
	dto.CHLagSeconds = lagSec
	return finalizeDataFreshness(dto)
}

func CampaignDashboardFreshness(now time.Time, usedCHMoney bool, chLag time.Duration, chAvailable bool) DataFreshnessDTO {
	chStale, lagSec := database.Freshness(chLag, 5*time.Minute)
	sources := []DataSourceFreshnessDTO{
		{Name: "counts", Consistency: "strong", Stale: false},
	}
	if usedCHMoney && chAvailable {
		sources = append(sources, DataSourceFreshnessDTO{
			Name:         "money",
			Consistency:  "eventual",
			Stale:        chStale,
			CHLagSeconds: lagSec,
		})
		return finalizeDataFreshness(DataFreshnessDTO{
			AsOf:         now.Format(time.RFC3339),
			Consistency:  "mixed",
			Stale:        chStale,
			CHLagSeconds: lagSec,
			Sources:      sources,
		})
	}
	sources = append(sources, DataSourceFreshnessDTO{
		Name:        "money",
		Consistency: "strong",
		Stale:       false,
	})
	return finalizeDataFreshness(DataFreshnessDTO{
		AsOf:        now.Format(time.RFC3339),
		Consistency: "strong",
		Stale:       !chAvailable,
		Sources:     sources,
	})
}

func WriteBuyerFraudExportPreamble(w *csv.Writer, freshness DataFreshnessDTO) error {
	disclaimer := "Buyer summary export; category labels only, no raw fraud_reason or placement identifiers."
	if freshness.Stale {
		disclaimer += " signals_degraded=true"
	}
	if err := w.Write([]string{"# disclaimer", disclaimer}); err != nil {
		return err
	}
	if freshness.Stale {
		return w.Write([]string{"# signals_degraded", "true"})
	}
	return nil
}
