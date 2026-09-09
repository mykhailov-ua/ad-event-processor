package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTimezoneAttestationMode_knownValues(t *testing.T) {
	require.Equal(t, TimezoneAttestationModeOff, ParseTimezoneAttestationMode("off"))
	require.Equal(t, TimezoneAttestationModeIPCountry, ParseTimezoneAttestationMode("ip_country"))
	require.Equal(t, TimezoneAttestationModeCampaignTarget, ParseTimezoneAttestationMode("campaign_target"))
	require.Equal(t, TimezoneAttestationModeStrict, ParseTimezoneAttestationMode("strict"))
	require.Equal(t, TimezoneAttestationModeStrict, ParseTimezoneAttestationMode(" STRICT "))
}

func TestParseTimezoneAttestationMode_emptyAndUnknown(t *testing.T) {
	require.Equal(t, TimezoneAttestationMode(""), ParseTimezoneAttestationMode(""))
	require.Equal(t, TimezoneAttestationMode(""), ParseTimezoneAttestationMode("bogus"))
}

func TestTimezoneAttestationMode_Effective_holdout(t *testing.T) {
	require.Equal(t, TimezoneAttestationModeIPCountry, TimezoneAttestationMode("").Effective())
	require.Equal(t, TimezoneAttestationModeOff, TimezoneAttestationModeOff.Effective())
	require.Equal(t, TimezoneAttestationModeCampaignTarget, TimezoneAttestationModeCampaignTarget.Effective())
}
