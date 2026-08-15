package ingestion

import (
	"sync/atomic"

	"github.com/bidshard/ad-event-processor/internal/config"
)

var (
	ortbScanMaxBytes   atomic.Int32
	ortbMaxQuoteChecks atomic.Int32
)

func init() {
	ortbScanMaxBytes.Store(int32(OrtbScanMaxBytes))
	ortbMaxQuoteChecks.Store(int32(OrtbMaxQuoteChecks))
}

func configureOrtbScanLimits(cfg *config.Config) {
	if cfg == nil {
		ortbScanMaxBytes.Store(int32(OrtbScanMaxBytes))
		ortbMaxQuoteChecks.Store(int32(OrtbMaxQuoteChecks))
		return
	}
	if cfg.OrtbScanMaxBytes > 0 {
		ortbScanMaxBytes.Store(int32(cfg.OrtbScanMaxBytes))
	} else {
		ortbScanMaxBytes.Store(int32(OrtbScanMaxBytes))
	}
	if cfg.OrtbMaxQuoteChecks > 0 {
		ortbMaxQuoteChecks.Store(int32(cfg.OrtbMaxQuoteChecks))
	} else {
		ortbMaxQuoteChecks.Store(int32(OrtbMaxQuoteChecks))
	}
}

func ortbScanMaxBytesLimit() int {
	return int(ortbScanMaxBytes.Load())
}

func ortbMaxQuoteChecksLimit() int {
	return int(ortbMaxQuoteChecks.Load())
}
