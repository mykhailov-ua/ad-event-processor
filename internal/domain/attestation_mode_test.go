package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveAttestationMode_legacyEnabledStrict(t *testing.T) {
	require.Equal(t, AttestationModeStrict, ResolveAttestationMode(AttestationModeOff, true))
}

func TestResolveAttestationMode_lightWinsOverLegacyOff(t *testing.T) {
	require.Equal(t, AttestationModeLight, ResolveAttestationMode(AttestationModeLight, false))
}

func TestAttestationMode_RequiresProbe_holdout(t *testing.T) {
	require.False(t, AttestationModeOff.RequiresProbe())
	require.True(t, AttestationModeLight.RequiresProbe())
	require.True(t, AttestationModeStrict.RequiresProbe())
}
