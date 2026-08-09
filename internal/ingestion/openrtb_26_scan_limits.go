package ingestion

import "espx/internal/config"

var (
	ortbScanMaxBytes   = OrtbScanMaxBytes
	ortbMaxQuoteChecks = OrtbMaxQuoteChecks
)

func configureOrtbScanLimits(cfg *config.Config) {
	if cfg == nil {
		ortbScanMaxBytes = OrtbScanMaxBytes
		ortbMaxQuoteChecks = OrtbMaxQuoteChecks
		return
	}
	if cfg.OrtbScanMaxBytes > 0 {
		ortbScanMaxBytes = cfg.OrtbScanMaxBytes
	} else {
		ortbScanMaxBytes = OrtbScanMaxBytes
	}
	if cfg.OrtbMaxQuoteChecks > 0 {
		ortbMaxQuoteChecks = cfg.OrtbMaxQuoteChecks
	} else {
		ortbMaxQuoteChecks = OrtbMaxQuoteChecks
	}
}
