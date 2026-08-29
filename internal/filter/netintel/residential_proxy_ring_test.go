package netintel

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestResidentialProxyRing_observeBuildsFarmPattern(t *testing.T) {
	ring := NewResidentialProxyRing()
	cid := uuid.New()
	campaignHash := domain.CRC32Castagnoli(&cid)
	now := monotonicNano()

	for i := range 271 {
		ring.Observe(campaignHash, false,
			HashResidentialProxyUser("imp-u"+itoaResidential(i%32)),
			HashResidentialProxyUA("ua-"+itoaResidential(i%11)), now)
	}
	for i := range 4 {
		ring.Observe(campaignHash, true,
			HashResidentialProxyUser("clk-u"+itoaResidential(i)),
			HashResidentialProxyUA("ua-"+itoaResidential(i%11)), now)
	}
	row, signal := ring.Observe(campaignHash, false,
		HashResidentialProxyUser("imp-u0"),
		HashResidentialProxyUA("ua-0"), now)
	require.True(t, signal, "row=%+v", row)
	require.GreaterOrEqual(t, row.UniqueUsers, 20)
	require.GreaterOrEqual(t, row.UniqueUAs, 11)
}

func itoaResidential(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
