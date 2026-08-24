package controlplane

import (
	"context"
	"net/http"
	"time"

	"ad-event-processor/internal/edge"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type PeriodDTO struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Timezone string `json:"timezone,omitempty"`
}

type DataFreshnessDTO struct {
	AsOf         string                   `json:"as_of"`
	Consistency  string                   `json:"consistency"`
	Stale        bool                     `json:"stale"`
	CHLagSeconds int                      `json:"ch_lag_seconds,omitempty"`
	Sources      []DataSourceFreshnessDTO `json:"sources,omitempty"`
}

type MetricsBlockDTO struct {
	SpendMicro   int64            `json:"spend_micro"`
	RevenueMicro int64            `json:"revenue_micro"`
	ProfitMicro  int64            `json:"profit_micro"`
	Conversions  int64            `json:"conversions"`
	CPAMicro     int64            `json:"cpa_micro"`
	ROIPct       float64          `json:"roi_pct"`
	Freshness    DataFreshnessDTO `json:"freshness"`
}

type ActionDTO struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	RequiresConfirm bool   `json:"requires_confirm"`
	ImpactMicro     int64  `json:"impact_micro,omitempty"`
}

type SourceRowDTO struct {
	CampaignID   string      `json:"campaign_id"`
	Sub1         string      `json:"sub1,omitempty"`
	Sub2         string      `json:"sub2,omitempty"`
	Country      string      `json:"country,omitempty"`
	Impressions  int64       `json:"impressions"`
	Clicks       int64       `json:"clicks"`
	Conversions  int64       `json:"conversions"`
	SpendMicro   int64       `json:"spend_micro"`
	RevenueMicro int64       `json:"revenue_micro"`
	ProfitMicro  int64       `json:"profit_micro"`
	CPAMicro     int64       `json:"cpa_micro"`
	ROIPct       float64     `json:"roi_pct"`
	CTR          float64     `json:"ctr"`
	IVTRate      float64     `json:"ivt_rate"`
	QualityScore float64     `json:"quality_score"`
	Actions      []ActionDTO `json:"actions,omitempty"`
}

type BuyerAttentionDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type BuyerCampaignPortfolioRowDTO struct {
	ID                      string  `json:"id"`
	Name                    string  `json:"name"`
	Status                  string  `json:"status"`
	PacingMode              string  `json:"pacing_mode"`
	Impressions7d           int64   `json:"impressions_7d"`
	Clicks7d                int64   `json:"clicks_7d"`
	SpendMicro              int64   `json:"spend_micro,omitempty"`
	BudgetMicro             int64   `json:"budget_micro,omitempty"`
	UtilizationPct          float64 `json:"utilization_pct,omitempty"`
	PacingDriftPct          float64 `json:"pacing_drift_pct,omitempty"`
	EstimatedPacingDriftPct float64 `json:"estimated_pacing_drift_pct,omitempty"`
	OverspendRisk           bool    `json:"overspend_risk,omitempty"`
	MarginBreach            bool    `json:"margin_breach,omitempty"`
}

type BuyerPortfolioDTO struct {
	CustomerID      string                         `json:"customer_id"`
	Period          PeriodDTO                      `json:"period"`
	Active          int                            `json:"active"`
	Paused          int                            `json:"paused"`
	Archived        int                            `json:"archived"`
	Impressions7d   int64                          `json:"impressions_7d"`
	Clicks7d        int64                          `json:"clicks_7d"`
	OverspendCount  int                            `json:"overspend_count,omitempty"`
	KPIs            *MetricsBlockDTO               `json:"kpis,omitempty"`
	Recommendations []RecommendationCardDTO        `json:"recommendations,omitempty"`
	Alerts          []AlertCardDTO                 `json:"alerts,omitempty"`
	Attention       []BuyerAttentionDTO            `json:"attention"`
	Campaigns       []BuyerCampaignPortfolioRowDTO `json:"campaigns"`
}

type BuyerPortfolioReader interface {
	GetBuyerPortfolio(ctx context.Context, customerID uuid.UUID) (BuyerPortfolioDTO, error)
}

type CampaignDashboardReader interface {
	GetCampaignDashboard(ctx context.Context, campaignID uuid.UUID) (CampaignDashboardDTO, error)
}

type RoleDashboardReader interface {
	GetAdOpsDashboard(ctx context.Context, customerID uuid.UUID) (AdOpsDashboardDTO, error)
	GetCFODashboard(ctx context.Context, customerID uuid.UUID) (CFODashboardDTO, error)
	GetAccountantDashboard(ctx context.Context, customerID uuid.UUID) (AccountantDashboardDTO, error)
	GetFraudDashboard(ctx context.Context, customerID uuid.UUID) (FraudDashboardDTO, error)
}

type BuyerCampaignRowDTO struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Status         string      `json:"status"`
	SpendMicro     int64       `json:"spend_micro"`
	BudgetMicro    int64       `json:"budget_micro"`
	UtilizationPct float64     `json:"utilization_pct"`
	ROIPct         float64     `json:"roi_pct"`
	PacingDriftPct float64     `json:"pacing_drift_pct"`
	OverspendRisk  bool        `json:"overspend_risk"`
	Actions        []ActionDTO `json:"actions,omitempty"`
}

type RecommendationCardDTO struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	CampaignID  string      `json:"campaign_id,omitempty"`
	Title       string      `json:"title"`
	Detail      string      `json:"detail"`
	Confidence  float64     `json:"confidence"`
	ImpactMicro int64       `json:"impact_micro"`
	CreatedAt   string      `json:"created_at"`
	ExpiresAt   string      `json:"expires_at"`
	Actions     []ActionDTO `json:"actions,omitempty"`
}

type AlertCardDTO struct {
	ID     string `json:"id"`
	Level  string `json:"level"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Route  string `json:"route,omitempty"`
}

type BuyerDashboardDTO struct {
	CustomerID      string                  `json:"customer_id"`
	Period          PeriodDTO               `json:"period"`
	KPIs            MetricsBlockDTO         `json:"kpis"`
	Campaigns       []BuyerCampaignRowDTO   `json:"campaigns"`
	TopSources      []SourceRowDTO          `json:"top_sources"`
	WorstSources    []SourceRowDTO          `json:"worst_sources"`
	Alerts          []AlertCardDTO          `json:"alerts"`
	Recommendations []RecommendationCardDTO `json:"recommendations"`
}

type AccountantCloseDTO struct {
	CustomerID            string `json:"customer_id"`
	BillingMonth          string `json:"billing_month"`
	InvariantOK           bool   `json:"invariant_ok"`
	InvariantDeltaMicro   int64  `json:"invariant_delta_micro"`
	UnreconciledPostbacks int    `json:"unreconciled_postbacks"`
}

type CFOSummaryDTO struct {
	CustomerID string          `json:"customer_id"`
	Period     PeriodDTO       `json:"period"`
	KPIs       MetricsBlockDTO `json:"unit_economics"`
}

type CFODashboardDTO struct {
	CustomerID           string          `json:"customer_id"`
	Period               PeriodDTO       `json:"period"`
	KPIs                 MetricsBlockDTO `json:"kpis"`
	BilledMicro          int64           `json:"billed_micro"`
	ARAgingMicro         int64           `json:"ar_aging_micro"`
	FeeTotalMicro        int64           `json:"fee_total_micro"`
	DisputeExposureMicro int64           `json:"dispute_exposure_micro"`
}

type AdOpsHealthDTO struct {
	CustomerID string    `json:"customer_id"`
	Period     PeriodDTO `json:"period"`
}

type AdOpsDashboardDTO struct {
	CustomerID   string                `json:"customer_id"`
	Period       PeriodDTO             `json:"period"`
	KPIs         MetricsBlockDTO       `json:"kpis"`
	Campaigns    []BuyerCampaignRowDTO `json:"campaigns"`
	WorstSources []SourceRowDTO        `json:"worst_sources"`
}

type ExportJobStatusDTO struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Format string `json:"format,omitempty"`
}

type AccountantDashboardDTO struct {
	CustomerID string               `json:"customer_id"`
	Period     PeriodDTO            `json:"period"`
	Close      AccountantCloseDTO   `json:"close"`
	TaxCountry string               `json:"tax_country,omitempty"`
	TaxScheme  string               `json:"tax_scheme,omitempty"`
	TaxVATID   string               `json:"tax_vat_id,omitempty"`
	ExportJobs []ExportJobStatusDTO `json:"export_jobs"`
}

type FraudOverviewDTO struct {
	CustomerID string    `json:"customer_id"`
	Period     PeriodDTO `json:"period"`
}

type FraudDashboardDTO struct {
	CustomerID          string                 `json:"customer_id"`
	Period              PeriodDTO              `json:"period"`
	GhostIVTCampaigns   int                    `json:"ghost_ivt_campaigns"`
	LabelsPending       int                    `json:"labels_pending"`
	EdgeBlockedFraud    uint64                 `json:"edge_blocked_fraud"`
	MLActiveVersionID   string                 `json:"ml_active_version_id,omitempty"`
	MLArtifactHash      string                 `json:"ml_artifact_hash,omitempty"`
	MLPrecision         float64                `json:"ml_precision,omitempty"`
	MLRecall            float64                `json:"ml_recall,omitempty"`
	MLDriftDetected     bool                   `json:"ml_drift_detected,omitempty"`
	MLDriftSummary      string                 `json:"ml_drift_summary,omitempty"`
	MLEvalGeneratedAt   string                 `json:"ml_eval_generated_at,omitempty"`
	MLEvalStatus        string                 `json:"ml_eval_status,omitempty"`
	MLEvalStale         bool                   `json:"ml_eval_stale,omitempty"`
	MLLabelMethod       string                 `json:"ml_label_method,omitempty"`
	MLShardsConsistent  *bool                  `json:"ml_shards_consistent,omitempty"`
	FraudTierThresholds FraudTierThresholdsDTO `json:"fraud_tier_thresholds"`
	GeoHints            []FraudGeoHintDTO      `json:"geo_hints,omitempty"`
	RecentLabels        []MLManualLabelDTO     `json:"recent_labels,omitempty"`
}

type FraudTierThresholdsDTO struct {
	Scope      string `json:"scope,omitempty"`
	PassMax    int    `json:"pass_max"`
	SuspectMax int    `json:"suspect_max"`
	IVTMax     int    `json:"ivt_max"`
	BlockAbove int    `json:"block_above"`
}

type FraudGeoHintDTO struct {
	Country    string  `json:"country"`
	IVTRate    float64 `json:"ivt_rate"`
	IVTEvents  int64   `json:"ivt_events"`
	Clicks     int64   `json:"clicks"`
	CampaignID string  `json:"campaign_id,omitempty"`
}

type EdgeMetricsPanelDTO struct {
	UpdatedAt      string            `json:"updated_at,omitempty"`
	IngressH1      uint64            `json:"ingress_h1"`
	IngressH2      uint64            `json:"ingress_h2"`
	IngressH3      uint64            `json:"ingress_h3"`
	BodyStream     uint64            `json:"body_stream"`
	BodyPeek       uint64            `json:"body_peek"`
	BodyRead       uint64            `json:"body_read"`
	Blocked        map[string]uint64 `json:"blocked"`
	TarpitTotal    uint64            `json:"tarpit_total"`
	BlacklistStale uint64            `json:"blacklist_stale"`
}

type OperatorDashboardDTO struct {
	Period PeriodDTO           `json:"period"`
	XDP    XDPPanelDTO         `json:"xdp"`
	Edge   EdgeMetricsPanelDTO `json:"edge"`
}

type XDPPanelDTO struct {
	UpdatedAt     string            `json:"updated_at,omitempty"`
	Pass          uint64            `json:"pass"`
	PassAllowlist uint64            `json:"pass_allowlist"`
	Fingerprints  uint64            `json:"fingerprints"`
	Drops         map[string]uint64 `json:"drops"`
}

type CampaignDashboardDTO struct {
	CampaignID string           `json:"campaign_id"`
	KPIs       MetricsBlockDTO  `json:"kpis"`
	Freshness  DataFreshnessDTO `json:"freshness"`
}

type DashboardsHTTPHandlers struct {
	BuyerPortfolio       BuyerPortfolioReader
	CampaignDashboard    CampaignDashboardReader
	RoleDashboards       RoleDashboardReader
	ReportJobs           *ReportJobRunner
	ApplyRateLimit       func(http.HandlerFunc) http.HandlerFunc
	RequirePermission    func(string, http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission func([]string, http.HandlerFunc) http.HandlerFunc
	ResolveCustomerID    func(*http.Request, *uuid.UUID) (uuid.UUID, error)
	WriteServiceError    func(http.ResponseWriter, error)
	XDPStatsReader       func(context.Context) (edge.Snapshot, error)
	EdgeMetricsReader    func(context.Context) (EdgeMetricsPanelDTO, error)
}

func (dashboards *DashboardsHTTPHandlers) Register(mux *http.ServeMux) {
	if dashboards == nil {
		return
	}
	limit := dashboards.ApplyRateLimit
	perm := dashboards.RequirePermission
	permAny := dashboards.RequireAnyPermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	mux.HandleFunc("GET /api/v1/dashboards/operator", limit(perm("shards:read", dashboards.getOperatorDashboard)))
	if dashboards.CampaignDashboard != nil {
		mux.HandleFunc("GET /api/v1/dashboards/campaign/{id}", limit(permAny([]string{"campaigns:read", "campaigns:read:masked"}, dashboards.getCampaignDashboard)))
	} else {
		mux.HandleFunc("GET /api/v1/dashboards/campaign/{id}", limit(perm("campaigns:read", dashboards.getCampaignDashboard)))
	}
	if dashboards.BuyerPortfolio != nil {
		mux.HandleFunc("GET /api/v1/dashboards/buyer", limit(permAny([]string{"campaigns:read", "campaigns:read:masked"}, dashboards.getBuyerDashboard)))
	}
	if dashboards.RoleDashboards != nil {
		mux.HandleFunc("GET /api/v1/dashboards/adops", limit(perm("campaigns:read", dashboards.getAdOpsDashboard)))
		mux.HandleFunc("GET /api/v1/dashboards/cfo", limit(perm("customers:read", dashboards.getCFODashboard)))
		mux.HandleFunc("GET /api/v1/dashboards/accountant", limit(perm("customers:read", dashboards.getAccountantDashboard)))
		mux.HandleFunc("GET /api/v1/dashboards/fraud", limit(perm("audit:read", dashboards.getFraudDashboard)))
	}
}

func (dashboards *DashboardsHTTPHandlers) getBuyerDashboard(w http.ResponseWriter, r *http.Request) {
	if dashboards.BuyerPortfolio == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "buyer dashboard unavailable")
		return
	}

	var customerID uuid.UUID
	if custStr := r.URL.Query().Get("customer_id"); custStr != "" {
		id, err := uuid.Parse(custStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return
		}
		customerID = id
	}
	if dashboards.ResolveCustomerID != nil {
		resolved, err := dashboards.ResolveCustomerID(r, nonNilUUID(customerID))
		if err != nil {
			dashboards.writeServiceError(w, err)
			return
		}
		customerID = resolved
	}
	if customerID == uuid.Nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id is required")
		return
	}

	resp, err := dashboards.BuyerPortfolio.GetBuyerPortfolio(r.Context(), customerID)
	if err != nil {
		dashboards.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (dashboards *DashboardsHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if dashboards.WriteServiceError != nil {
		dashboards.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", err.Error())
}

func (dashboards *DashboardsHTTPHandlers) getCampaignDashboard(w http.ResponseWriter, r *http.Request) {
	campaignIDStr := r.PathValue("id")
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}

	if dashboards.CampaignDashboard != nil {
		resp, err := dashboards.CampaignDashboard.GetCampaignDashboard(r.Context(), campaignID)
		if err != nil {
			dashboards.writeServiceError(w, err)
			return
		}
		httpresponse.JSON(w, http.StatusOK, resp)
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
	if dashboards != nil && dashboards.EdgeMetricsReader != nil {
		if edge, err := dashboards.EdgeMetricsReader(r.Context()); err == nil {
			resp.Edge = edge
		}
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (dashboards *DashboardsHTTPHandlers) resolveRoleCustomerID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	var customerID uuid.UUID
	if custStr := r.URL.Query().Get("customer_id"); custStr != "" {
		id, err := uuid.Parse(custStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return uuid.Nil, false
		}
		customerID = id
	}
	if dashboards.ResolveCustomerID != nil {
		resolved, err := dashboards.ResolveCustomerID(r, nonNilUUID(customerID))
		if err != nil {
			dashboards.writeServiceError(w, err)
			return uuid.Nil, false
		}
		customerID = resolved
	}
	if customerID == uuid.Nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id is required")
		return uuid.Nil, false
	}
	return customerID, true
}

func (dashboards *DashboardsHTTPHandlers) getAdOpsDashboard(w http.ResponseWriter, r *http.Request) {
	if dashboards.RoleDashboards == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "adops dashboard unavailable")
		return
	}
	customerID, ok := dashboards.resolveRoleCustomerID(w, r)
	if !ok {
		return
	}
	resp, err := dashboards.RoleDashboards.GetAdOpsDashboard(r.Context(), customerID)
	if err != nil {
		dashboards.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (dashboards *DashboardsHTTPHandlers) getCFODashboard(w http.ResponseWriter, r *http.Request) {
	if dashboards.RoleDashboards == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "cfo dashboard unavailable")
		return
	}
	customerID, ok := dashboards.resolveRoleCustomerID(w, r)
	if !ok {
		return
	}
	resp, err := dashboards.RoleDashboards.GetCFODashboard(r.Context(), customerID)
	if err != nil {
		dashboards.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (dashboards *DashboardsHTTPHandlers) getAccountantDashboard(w http.ResponseWriter, r *http.Request) {
	if dashboards.RoleDashboards == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "accountant dashboard unavailable")
		return
	}
	customerID, ok := dashboards.resolveRoleCustomerID(w, r)
	if !ok {
		return
	}
	resp, err := dashboards.RoleDashboards.GetAccountantDashboard(r.Context(), customerID)
	if err != nil {
		dashboards.writeServiceError(w, err)
		return
	}
	if dashboards.ReportJobs != nil {
		jobs := dashboards.ReportJobs.ListJobsByCustomer(customerID.String(), 8)
		resp.ExportJobs = make([]ExportJobStatusDTO, 0, len(jobs))
		for _, job := range jobs {
			resp.ExportJobs = append(resp.ExportJobs, ExportJobStatusDTO{
				ID:     job.ID,
				Status: job.Status,
				Format: job.Format,
			})
		}
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (dashboards *DashboardsHTTPHandlers) getFraudDashboard(w http.ResponseWriter, r *http.Request) {
	if dashboards.RoleDashboards == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "fraud dashboard unavailable")
		return
	}
	customerID, ok := dashboards.resolveRoleCustomerID(w, r)
	if !ok {
		return
	}
	resp, err := dashboards.RoleDashboards.GetFraudDashboard(r.Context(), customerID)
	if err != nil {
		dashboards.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}
