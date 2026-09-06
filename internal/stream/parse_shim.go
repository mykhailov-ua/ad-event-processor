package stream

import (
	"ad-event-processor/internal/filter"

	"github.com/google/uuid"
)

func ParseUUID(b []byte, dst *uuid.UUID) bool {
	return filter.ParseUUID(b, dst)
}
