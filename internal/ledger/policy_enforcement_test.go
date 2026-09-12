package ledger

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCampaignBreachEnforcement_defaultsPauseCampaign_holdout(t *testing.T) {
	assert.Equal(t, EnforcementPauseCampaign, CampaignBreachEnforcement(&Policy{}))
	assert.Equal(t, EnforcementPauseCampaign, CampaignBreachEnforcement(&Policy{Enforcement: EnforcementBlacklistPlacement}))
}

func TestPlacementBreachEnforcement_defaultsBlacklist_holdout(t *testing.T) {
	assert.Equal(t, EnforcementBlacklistPlacement, PlacementBreachEnforcement(&Policy{}))
	assert.Equal(t, EnforcementPauseCampaign, PlacementBreachEnforcement(&Policy{Enforcement: EnforcementPauseCampaign}))
}

func TestPolicyCooldownSec_clamps_holdout(t *testing.T) {
	assert.Equal(t, defaultMarginGuardCooldownSec, PolicyCooldownSec(nil))
	assert.Equal(t, 120, PolicyCooldownSec(&Policy{CooldownSec: 120}))
	assert.Equal(t, maxMarginGuardCooldownSec, PolicyCooldownSec(&Policy{CooldownSec: 999999}))
}

func TestWithinCooldown_holdout(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	last := now.Add(-30 * time.Second)
	assert.True(t, WithinCooldown(last, 60, now))
	assert.False(t, WithinCooldown(last, 10, now))
	assert.False(t, WithinCooldown(last, 0, now))
}

func TestPolicyWantsPlatformPause_holdout(t *testing.T) {
	assert.False(t, PolicyWantsPlatformPause(&Policy{}))
	assert.True(t, PolicyWantsPlatformPause(&Policy{PlatformPause: true}))
	assert.True(t, PolicyWantsPlatformPause(&Policy{Enforcement: EnforcementPlatformPause}))
}
