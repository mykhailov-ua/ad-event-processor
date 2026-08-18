package licensing

import (
	"log/slog"
	"sync"
)

var (
	watermarkMu      sync.Mutex
	watermarkApplied bool
)

func UpdateLogWatermark(claims *LicenseClaims) {
	if claims == nil {
		return
	}
	watermarkMu.Lock()
	defer watermarkMu.Unlock()

	logger := slog.Default().With(
		"license_customer", claims.CustomerName,
		"license_deployment_id", claims.DeploymentID,
	)
	if claims.SKU != "" {
		logger = logger.With("license_sku", claims.SKU)
	}
	if claims.Plan != "" {
		logger = logger.With("license_plan", claims.Plan)
	}

	slog.SetDefault(logger)
	if !watermarkApplied {
		watermarkApplied = true
		slog.Info("license watermark applied")
	}
}
