package filter

import (
	"net/netip"
	"time"

	"ad-event-processor/internal/filter/netintel"

	"github.com/google/uuid"
)

type MockRegistry = mockRegistry

func ParseProxyVPNFeedLine(line string) (netip.Prefix, uint8, uint32, bool) {
	return netintel.ParseProxyVPNFeedLine(line)
}

func ProxyVPNConnTypeBlocks(connType uint8) bool {
	return netintel.ProxyVPNConnTypeBlocks(connType)
}

func HashResidentialProxyUser(s string) uint32 {
	return netintel.HashResidentialProxyUser(s)
}

func HashResidentialProxyUA(s string) uint32 {
	return netintel.HashResidentialProxyUA(s)
}

func AppendCampaignHashTag(dst []byte, id uuid.UUID) []byte {
	return appendCampaignHashTag(dst, id)
}

func BudgetQuotaKey(id uuid.UUID) string {
	return budgetQuotaKey(id)
}

func TimezoneMismatchHours(browserTZ, country string, now time.Time) (bool, int) {
	return netintel.TimezoneMismatchHours(browserTZ, country, now)
}
