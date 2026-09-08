package ingest

import (
	"net/http"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/moderatorcorpus"

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
	brandID := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.BrandID = &brandID
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
	store := clickHookBrandStore(t, brandID)
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, NewFilterEngine(0, filter), nil, nil, NewJumpHashSharder(1), "fraud-stream", store)
	h.ConfigureCIDR(cidrBlockTestTable(t, "203.0.113.0/24"))
	configureClickHookRotationTables(h)
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

func reviewCorpusPolicyHandler(t *testing.T, filter EventFilter) (*AdsPacketHandler, uuid.UUID) {
	t.Helper()
	if filter == nil {
		filter = &countingFilter{}
	}
	cid := uuid.New()
	brandID := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.BrandID = &brandID
		c.ReviewTrafficAction = domain.ReviewTrafficActionSafePage
		c.CIDRBlockEnabled = false
	})
	t.Cleanup(func() {
		lockStaticCampaign(func(c *domain.Campaign) {
			c.CIDRBlockEnabled = false
			c.ReviewTrafficAction = domain.ReviewTrafficActionSafePage
		})
		cachedMockCamp.Store(nil)
	})
	cachedMockCamp.Store(nil)

	cfg := &config.Config{MaxRequestBodySize: 1 << 20, ModeratorCorpusEnabled: true}
	store := clickHookBrandStore(t, brandID)
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, NewFilterEngine(0, filter), nil, nil, NewJumpHashSharder(1), "fraud-stream", store)
	configureClickHookRotationTables(h)
	return h, cid
}

func TestReviewTrafficPolicy_moderatorCorpusMatch(t *testing.T) {
	h, cid := reviewCorpusPolicyHandler(t, nil)
	ja3 := "771,4865-4866-4867"
	entry, err := moderatorcorpus.EntryFromTuple(moderatorcorpus.Tuple{JA3: ja3})
	require.NoError(t, err)
	table := NewModeratorCorpusTable()
	table.Publish(BuildModeratorCorpusSnapshot([]moderatorcorpus.Entry{entry}, 1))
	h.ConfigureModeratorCorpus(table)

	conn := serveClickWithJA3(h, cid, "8.8.8.8", ja3)
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	require.Contains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: moderator_corpus")
}

func TestReviewTrafficPolicy_moderatorCorpusEmptyHoldout(t *testing.T) {
	filter := &countingFilter{}
	h, cid := reviewCorpusPolicyHandler(t, filter)

	conn := serveClickWithJA3(h, cid, "8.8.8.8", "771,4865-4866-4867")
	require.NotContains(t, string(conn.Written()), "X-ad-event-processor-Safe-View: moderator_corpus")
	require.Equal(t, 1, filter.calls)
}
