package ingest

import (
	"net/netip"
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/moderatorintel"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModeratorIntelHook_campaignFlagGate(t *testing.T) {
	t.Parallel()
	prefix, err := netip.ParsePrefix("203.0.113.0/24")
	require.NoError(t, err)
	table := NewModeratorIPTable()
	table.PublishEntriesForTest([]moderatorintel.Entry{{Prefix: prefix, Network: 1}}, 1)
	cid := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.ModeratorIntelEnabled = true
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) { c.ModeratorIntelEnabled = false })
		cachedMockCamp.Store(nil)
	})
	cachedMockCamp.Store(nil)
	h := &AdsPacketHandler{
		registry:         &mockRegistry{},
		moderatorIPTable: table,
		moderatorMetrics: newModeratorIntelMetrics(),
	}
	match, _ := h.moderatorIPShouldSafeView("203.0.113.44", cid)
	assert.True(t, match)
	lockStaticCampaign(func(c *domain.Campaign) { c.ModeratorIntelEnabled = false })
	cachedMockCamp.Store(nil)
	match, _ = h.moderatorIPShouldSafeView("203.0.113.44", cid)
	assert.False(t, match)
}
