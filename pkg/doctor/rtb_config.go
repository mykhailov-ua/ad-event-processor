package doctor

import (
	"context"
	"fmt"
	"time"
)

type RtbConfigProbe struct {
	Deps ProbeDeps
}

func (RtbConfigProbe) Name() string { return "rtb_config" }

func (p RtbConfigProbe) Run(ctx context.Context) Result {
	start := time.Now()
	latency := func() int64 { return time.Since(start).Milliseconds() }
	cfg := p.Deps.Config
	if cfg == nil {
		return Result{Name: "rtb_config", Status: StatusSkip, Detail: "config not loaded", Latency: latency()}
	}
	if !cfg.RtbEnabled() {
		return Result{Name: "rtb_config", Status: StatusSkip, Detail: "RTB disabled", Latency: latency()}
	}
	_ = ctx

	var issues []string
	if cfg.RtbMode == "live" && cfg.RtbExchangeMaxQPS <= 0 {
		issues = append(issues, "RTB_EXCHANGE_MAX_QPS=0 in live mode")
	}
	if cfg.RtbDealOutcomeFlushMs <= 0 {
		issues = append(issues, "RTB_DEAL_OUTCOME_FLUSH_MS unset")
	}
	if cfg.IsClickHouseEnabled() {
		if !cfg.ClickHouseJanitorEnabled {
			issues = append(issues, "CH_JANITOR_ENABLED=false")
		}
		if cfg.ClickHouseRetentionDaysRtbDealOutcomes <= 0 {
			issues = append(issues, "CH_RETENTION_DAYS_RTB_DEAL_OUTCOMES unset")
		}
		if cfg.ClickHouseRetentionDaysRtbExchangeLog <= 0 {
			issues = append(issues, "CH_RETENTION_DAYS_RTB_EXCHANGE_LOG unset")
		}
	} else if cfg.RtbMode != "off" {
		issues = append(issues, "CH disabled while RTB enabled (exchange log / reconcile need CH)")
	}
	if len(issues) == 0 {
		return Result{
			Name:    "rtb_config",
			Status:  StatusPass,
			Detail:  fmt.Sprintf("mode=%s janitor=%v flush_ms=%d", cfg.RtbMode, cfg.ClickHouseJanitorEnabled, cfg.RtbDealOutcomeFlushMs),
			Latency: latency(),
		}
	}
	return Result{
		Name:    "rtb_config",
		Status:  StatusWarn,
		Detail:  issues[0],
		Latency: latency(),
	}
}
