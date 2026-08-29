package ingest

import (
	"net/http"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func reviewPolicyHandler(t *testing.T, action domain.ReviewTrafficAction, filter *countingFilter) (*AdsPacketHandler, uuid.UUID) {
	t.Helper()
	if filter == nil {
		filter = &countingFilter{}
	}
	cid := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.ReviewTrafficAction = action
		c.CIDRBlockEnabled = true
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) {
			c.ReviewTrafficAction = domain.ReviewTrafficActionSafePage
			c.CIDRBlockEnabled = false
		})
		cachedMockCamp.Store(nil)
	})
	cachedMockCamp.Store(nil)

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, NewFilterEngine(0, filter), nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	h.ConfigureCIDR(cidrBlockTestTable(t, "203.0.113.0/24"))
	return h, cid
}

func TestReviewTrafficPolicy_blockAction(t *testing.T) {
	h, cid := reviewPolicyHandler(t, domain.ReviewTrafficActionBlock, nil)
	conn := serveClickFromIP(h, cid, "203.0.113.44")
	require.Equal(t, http.StatusForbidden, ParseGnetHTTPStatus(conn.Written()))
	assert.Contains(t, string(conn.Written()), "review traffic blocked")
}

func TestReviewTrafficPolicy_passthroughContinues(t *testing.T) {
	filter := &countingFilter{}
	h, cid := reviewPolicyHandler(t, domain.ReviewTrafficActionPassthrough, filter)
	conn := serveClickFromIP(h, cid, "203.0.113.44")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View")
	require.Equal(t, 1, filter.calls)
}

func TestReviewTrafficPolicy_defaultSafePage(t *testing.T) {
	h, cid := reviewPolicyHandler(t, domain.ReviewTrafficActionSafePage, nil)
	conn := serveClickFromIP(h, cid, "203.0.113.44")
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	require.Contains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: l1")
}
