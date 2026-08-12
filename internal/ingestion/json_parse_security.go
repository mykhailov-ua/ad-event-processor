package ingestion

import "github.com/bidshard/ad-event-processor/internal/config"

var jsonStrictUTF8Enabled = true

func configureJSONParseSecurity(cfg *config.Config) {
	if cfg == nil {
		jsonStrictUTF8Enabled = true
		return
	}
	jsonStrictUTF8Enabled = cfg.JSONStrictUTF8
}
