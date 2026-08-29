package ingest

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type stubCampaignRegistry struct {
	camp *domain.Campaign
	ok   bool
}

func (s stubCampaignRegistry) Exists(uuid.UUID) bool { return s.ok }
func (s stubCampaignRegistry) Add(uuid.UUID, uuid.UUID, *uuid.UUID, string, domain.PacingMode, int64, string, int32, int32, []string) {
}
func (s stubCampaignRegistry) GetCustomerID(uuid.UUID) (uuid.UUID, bool) { return uuid.Nil, false }
func (s stubCampaignRegistry) GetCampaign(uuid.UUID) (*domain.Campaign, bool) {
	return s.camp, s.ok
}
func (s stubCampaignRegistry) Sync(context.Context) (int, error)        { return 0, nil }
func (s stubCampaignRegistry) StartSync(context.Context, time.Duration) {}
func (s stubCampaignRegistry) Wait(context.Context) error               { return nil }

func TestSafePageEligibleReject(t *testing.T) {
	assert.True(t, safePageEligibleReject(filterRejectFraud))
	assert.True(t, safePageEligibleReject(filterRejectPlacementBlocked))
	assert.False(t, safePageEligibleReject(filterRejectBudget))
}

func TestResolveSafePageLanding(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	reg := stubCampaignRegistry{
		ok: true,
		camp: &domain.Campaign{
			ID:              id,
			SafePageURL:     "https://safe.example/",
			SafePageEnabled: true,
		},
	}
	url, ok := resolveSafePageLanding(reg, id)
	assert.True(t, ok)
	assert.Equal(t, "https://safe.example/", url)

	reg.camp.SafePageEnabled = false
	_, ok = resolveSafePageLanding(reg, id)
	assert.False(t, ok)
}

func TestResolveSafePageAction_placementBlocked(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	reg := stubCampaignRegistry{
		ok: true,
		camp: &domain.Campaign{
			ID:              id,
			SafePageURL:     "https://safe.example/lp",
			SafePageEnabled: true,
		},
	}
	out := trackOutcome{
		Status:     trackStatusRejected,
		RejectKind: filterRejectPlacementBlocked,
	}
	action, url := resolveSafePageAction(reg, id, out, false)
	assert.Equal(t, safePageActionInPlace, action)
	assert.Equal(t, "https://safe.example/lp", url)
}

func TestResolveSafePageAction_forceSafe(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	reg := stubCampaignRegistry{
		ok: true,
		camp: &domain.Campaign{
			ID:              id,
			SafePageURL:     "https://safe.example/lp",
			SafePageEnabled: true,
		},
	}
	out := trackOutcome{Status: trackStatusAccepted}
	action, url := resolveSafePageAction(reg, id, out, true)
	assert.Equal(t, safePageActionInPlace, action)
	assert.Equal(t, "https://safe.example/lp", url)
}

func TestResolveSafePageAction_cleanAccepted(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	reg := stubCampaignRegistry{
		ok: true,
		camp: &domain.Campaign{
			ID:              id,
			SafePageURL:     "https://safe.example/lp",
			SafePageEnabled: true,
		},
	}
	out := trackOutcome{Status: trackStatusAccepted}
	action, url := resolveSafePageAction(reg, id, out, false)
	assert.Equal(t, safePageActionNone, action)
	assert.Empty(t, url)
}

func TestResolveSafePageAction_deliveryRedirect(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	reg := stubCampaignRegistry{
		ok: true,
		camp: &domain.Campaign{
			ID:              id,
			SafePageURL:     "https://safe.example/lp",
			SafePageEnabled: true,
		},
	}
	out := trackOutcome{
		Status:     trackStatusRejected,
		RejectKind: filterRejectPlacementBlocked,
	}
	action, url := resolveSafePageActionDelivery(reg, id, out, false, safePageDeliveryRedirect)
	assert.Equal(t, safePageActionRedirect, action)
	assert.Equal(t, "https://safe.example/lp", url)
}

func TestParseSafePageStubCampaignID(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	parsed, ok := parseSafePageStubCampaignID([]byte("/safe_page_stub?campaign_id=550e8400-e29b-41d4-a716-446655440000"))
	assert.True(t, ok)
	assert.Equal(t, id, parsed)
}

func TestHttp1MatchForceSafeHeader(t *testing.T) {
	assert.True(t, http1MatchForceSafeHeader([]byte("X-ad-event-processor-Force-Safe")))
	assert.False(t, http1MatchForceSafeHeader([]byte("X-ad-event-processor-Safe-Page")))
}

func TestHttp1ForceSafeValue(t *testing.T) {
	assert.True(t, http1ForceSafeValue([]byte("1")))
	assert.True(t, http1ForceSafeValue([]byte("true")))
	assert.False(t, http1ForceSafeValue([]byte("0")))
}
