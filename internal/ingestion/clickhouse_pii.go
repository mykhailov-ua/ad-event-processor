package ingestion

import (
	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/piihash"
)

type clickhousePIIFields struct {
	ipHash      [16]byte
	uaHash      [16]byte
	userIDHash  [16]byte
	subnetHash  [16]byte
	saltVersion uint8
}

func hashEventPII(h *piihash.Hasher, e *domain.Event) clickhousePIIFields {
	if h == nil || e == nil {
		return clickhousePIIFields{}
	}
	return clickhousePIIFields{
		ipHash:      h.HashIP(e.IP),
		uaHash:      h.HashUA(e.UA),
		userIDHash:  h.HashUserID(e.UserID),
		subnetHash:  h.HashSubnet(e.IP),
		saltVersion: h.Version(),
	}
}
