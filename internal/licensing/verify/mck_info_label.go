package verify

import "strings"

const DefaultMCKInfoLabel = "license-mck-v2"

var (
	buildMCKInfoLabel      string
	mckInfoLabelOverride   string
	mckInfoLabelOverrideOn bool
)

func MCKInfoLabel() string {
	if mckInfoLabelOverrideOn && strings.TrimSpace(mckInfoLabelOverride) != "" {
		return strings.TrimSpace(mckInfoLabelOverride)
	}
	if v := strings.TrimSpace(buildMCKInfoLabel); v != "" {
		return v
	}
	return DefaultMCKInfoLabel
}

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
