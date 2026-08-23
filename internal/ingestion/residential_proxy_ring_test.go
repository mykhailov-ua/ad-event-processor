package ingestion

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestResidentialProxyRing_observeBuildsFarmPattern(t *testing.T) {
	ring := NewResidentialProxyRing()
	cid := uuid.New()
	campaignHash := crc32Castagnoli(&cid)
	now := monotonicNano()

	for i := 0; i < 271; i++ {
		ring.observe(campaignHash, false,
			hashResidentialProxyUser("imp-u"+itoaResidential(i%32)),
			hashResidentialProxyUA("ua-"+itoaResidential(i%11)), now)
	}
	for i := 0; i < 4; i++ {
		ring.observe(campaignHash, true,
			hashResidentialProxyUser("clk-u"+itoaResidential(i)),
			hashResidentialProxyUA("ua-"+itoaResidential(i%11)), now)
	}
	row, signal := ring.observe(campaignHash, false,
		hashResidentialProxyUser("imp-u0"),
		hashResidentialProxyUA("ua-0"), now)
	require.True(t, signal, "row=%+v", row)
	require.GreaterOrEqual(t, row.UniqueUsers, 20)
	require.GreaterOrEqual(t, row.UniqueUAs, 11)
}
