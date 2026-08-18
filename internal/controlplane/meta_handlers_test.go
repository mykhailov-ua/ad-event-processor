package controlplane

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMetaLicense_tierWarnings(t *testing.T) {
	validUntil := time.Now().Add(3 * 24 * time.Hour)
	lic := BuildMetaLicense(MetaLicenseBuildInput{
		State:         "ACTIVE",
		PlanCode:      "pro",
		ValidUntil:    validUntil,
		HasValidUntil: true,
		TierWarnings:  []string{"Approaching active campaign cap (8/10)."},
	})
	require.NotNil(t, lic)
	assert.Equal(t, "pro", lic.PlanCode)
	assert.GreaterOrEqual(t, lic.RenewDays, 2)
	assert.LessOrEqual(t, lic.RenewDays, 3)
	assert.Len(t, lic.TierWarnings, 1)
}

func TestBuildMetaLicense_emptyState(t *testing.T) {
	assert.Nil(t, BuildMetaLicense(MetaLicenseBuildInput{}))
}
