package ingestion

import (
	"sync/atomic"

	"ad-event-processor/internal/config"
)

var jsonStrictUTF8Enabled atomic.Bool

func init() {
	jsonStrictUTF8Enabled.Store(true)
}

func configureJSONParseSecurity(cfg *config.Config) {
	if cfg == nil {
		jsonStrictUTF8Enabled.Store(true)
		return
	}
	jsonStrictUTF8Enabled.Store(cfg.JSONStrictUTF8)
}

func jsonStrictUTF8() bool {
	return jsonStrictUTF8Enabled.Load()
}
