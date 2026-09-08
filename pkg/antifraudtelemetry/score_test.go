package antifraudtelemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScore_automationLeak_holdout(t *testing.T) {
	v := Score(Input{Webdriver: 1})
	require.True(t, v.AutomationLeak)
	v = Score(Input{AutomationLeak: AutomationLeakPlaywright})
	require.True(t, v.AutomationLeak)
}

func TestScore_templateBehavior_holdout(t *testing.T) {
	v := Score(Input{
		DwellMs:        3000,
		PointerCVMilli: 20,
		ScrollCVMilli:  18,
		ScrollJerkMilli: 12,
		FooterReachMs:  2200,
	})
	require.True(t, v.TemplateBehavior)
}

func TestScore_proxyJitter_holdout(t *testing.T) {
	v := Score(Input{
		RTTSamples:     []uint16{12, 14, 11, 58, 13, 15},
		ServerRTTSynMS: 14,
		ConnTimingSet:  1,
	})
	require.True(t, v.ProxyJitter)
}

func TestScore_residentialBaseline_pass(t *testing.T) {
	v := Score(Input{
		DwellMs:           12000,
		TrustedRatioMilli: 920,
		PointerCVMilli:    280,
		PointerDtCVMilli:  210,
		ScrollCVMilli:     190,
		RTTSamples:        []uint16{22, 24, 23, 25, 22},
	})
	require.False(t, v.AutomationLeak)
	require.False(t, v.TemplateBehavior)
	require.False(t, v.ProxyJitter)
	require.False(t, v.UntrustedEvents)
}

func TestScore_untrustedRequiresKinematics_holdout(t *testing.T) {
	v := Score(Input{TrustedRatioMilli: 200})
	require.False(t, v.UntrustedEvents)
	v = Score(Input{
		TrustedRatioMilli: 200,
		PointerDtCVMilli:  20,
		PointerCVMilli:    30,
	})
	require.True(t, v.UntrustedEvents)
}
