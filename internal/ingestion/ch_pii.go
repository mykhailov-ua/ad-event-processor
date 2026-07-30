package ingestion

import (
	"espx/internal/campaignmodel"
	"espx/pkg/piihash"
)

type chPIIFields struct {
	ipHash      [16]byte
	uaHash      [16]byte
	userIDHash  [16]byte
	subnetHash  [16]byte
	saltVersion uint8
}

func hashEventPII(h *piihash.Hasher, e *campaignmodel.Event) chPIIFields {
	if h == nil || e == nil {
		return chPIIFields{}
	}
	return chPIIFields{
		ipHash:      h.HashIP(e.IP),
		uaHash:      h.HashUA(e.UA),
		userIDHash:  h.HashUserID(e.UserID),
		subnetHash:  h.HashSubnet(e.IP),
		saltVersion: h.Version(),
	}
}
