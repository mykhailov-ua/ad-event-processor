package stream

import "ad-event-processor/internal/domain"

func conversionValidationPending(payload []byte) bool {
	return domain.ConversionValidationPending(payload)
}
