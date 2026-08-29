package fraud

import "ad-event-processor/internal/metrics"

var filterFraudStreamWriteErrors = metrics.FilterInternalErrors.WithLabelValues("fraud_stream_write")
