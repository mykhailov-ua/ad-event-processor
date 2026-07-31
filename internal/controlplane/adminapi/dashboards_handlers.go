package adminapi

import (
	"context"
	"net/http"
	"time"

	"espx/internal/edge/xdpstats"
	"espx/pkg/httpresponse"

	"github.com/google/uuid"
)

type DashboardsHTTPHandlers struct {
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
	XDPStatsReader    func(context.Context) (xdpstats.Snapshot, error)
}

func (dashboards *DashboardsHTTPHandlers) Register(mux *http.ServeMux) {
	if dashboards == nil {
		return
	}
	limit := dashboards.ApplyRateLimit
	perm := dashboards.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	mux.HandleFunc("GET /api/v1/dashboards/buyer", limit(perm("campaigns:read", dashboards.notImplemented)))
	mux.HandleFunc("GET /api/v1/dashboards/adops", limit(perm("campaigns:read", dashboards.notImplemented)))
	mux.HandleFunc("GET /api/v1/dashboards/accountant", limit(perm("customers:read", dashboards.notImplemented)))
	mux.HandleFunc("GET /api/v1/dashboards/cfo", limit(perm("customers:read", dashboards.notImplemented)))
	mux.HandleFunc("GET /api/v1/dashboards/fraud", limit(perm("audit:read", dashboards.notImplemented)))
	mux.HandleFunc("GET /api/v1/dashboards/operator", limit(perm("shards:read", dashboards.getOperatorDashboard)))
	mux.HandleFunc("GET /api/v1/dashboards/campaign/{id}", limit(perm("campaigns:read", dashboards.getCampaignDashboard)))
}

func (dashboards *DashboardsHTTPHandlers) getCampaignDashboard(w http.ResponseWriter, r *http.Request) {
	campaignIDStr := r.PathValue("id")
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}

	resp := CampaignDashboardDTO{
		CampaignID: campaignID.String(),
		KPIs: MetricsBlockDTO{
			SpendMicro:   150000000,
			RevenueMicro: 180000000,
			ProfitMicro:  30000000,
			Conversions:  120,
			CPAMicro:     1250000,
			ROIPct:       20.0,
			Freshness: DataFreshnessDTO{
				AsOf:         time.Now().UTC().Format(time.RFC3339),
				Consistency:  "eventual",
				Stale:        true,
				CHLagSeconds: 360,
			},
		},
		Freshness: DataFreshnessDTO{
			AsOf:         time.Now().UTC().Format(time.RFC3339),
			Consistency:  "eventual",
			Stale:        true,
			CHLagSeconds: 360,
		},
	}

	httpresponse.JSON(w, http.StatusOK, resp)
}

func (dashboards *DashboardsHTTPHandlers) getOperatorDashboard(w http.ResponseWriter, r *http.Request) {
	resp := OperatorDashboardDTO{
		Period: PeriodDTO{
			From: time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339),
			To:   time.Now().UTC().Format(time.RFC3339),
		},
	}
	if dashboards != nil && dashboards.XDPStatsReader != nil {
		if snap, err := dashboards.XDPStatsReader(r.Context()); err == nil {
			resp.XDP = XDPPanelDTO{
				UpdatedAt:     snap.UpdatedAt.UTC().Format(time.RFC3339),
				Pass:          snap.Pass,
				PassAllowlist: snap.PassAllow,
				Fingerprints:  snap.Fingerprints,
				Drops:         snap.Drops,
			}
		}
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (dashboards *DashboardsHTTPHandlers) notImplemented(w http.ResponseWriter, _ *http.Request) {
	httpresponse.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "dashboard stub; UI deferred — docs/DEVELOPMENT.md")
}
