package licensing

import "strings"

// DefaultMCKInfoLabel is the HKDF info string for DeriveMCK in dev/test builds.
// Release builds override via -X licensing.buildMCKInfoLabel in release_garble.sh.
const DefaultMCKInfoLabel = "license-mck-v2"

var (
	buildMCKInfoLabel      string
	mckInfoLabelOverride   string
	mckInfoLabelOverrideOn bool
)

// MCKInfoLabel returns the active HKDF info label for MCK derivation.
func MCKInfoLabel() string {
	if mckInfoLabelOverrideOn && strings.TrimSpace(mckInfoLabelOverride) != "" {
		return strings.TrimSpace(mckInfoLabelOverride)
	}
	if v := strings.TrimSpace(buildMCKInfoLabel); v != "" {
		return v
	}
	return DefaultMCKInfoLabel
}

// SetMCKInfoLabelForTest overrides MCKInfoLabel for the duration of a test.
func SetMCKInfoLabelForTest(label string) func() {
	prevOn := mckInfoLabelOverrideOn
	prev := mckInfoLabelOverride
	mckInfoLabelOverrideOn = true
	mckInfoLabelOverride = label
	return func() {
		mckInfoLabelOverrideOn = prevOn
		mckInfoLabelOverride = prev
	}
}
