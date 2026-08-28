package reports

import (
	"time"

	"ad-event-processor/internal/database"
)

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
		return dto
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
	return dto
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
		return DataFreshnessDTO{
			AsOf:         now.Format(time.RFC3339),
			Consistency:  "mixed",
			Stale:        chStale,
			CHLagSeconds: lagSec,
			Sources:      sources,
		}
	}
	sources = append(sources, DataSourceFreshnessDTO{
		Name:        "money",
		Consistency: "strong",
		Stale:       false,
	})
	return DataFreshnessDTO{
		AsOf:        now.Format(time.RFC3339),
		Consistency: "strong",
		Stale:       !chAvailable,
		Sources:     sources,
	}
}
