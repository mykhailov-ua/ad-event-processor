package domain

import "strings"

type ReviewTrafficAction string

const (
	ReviewTrafficActionSafePage    ReviewTrafficAction = "safe_page"
	ReviewTrafficActionBlock       ReviewTrafficAction = "block"
	ReviewTrafficActionPassthrough ReviewTrafficAction = "passthrough"
)

func ParseReviewTrafficAction(raw string) ReviewTrafficAction {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case string(ReviewTrafficActionBlock):
		return ReviewTrafficActionBlock
	case string(ReviewTrafficActionPassthrough):
		return ReviewTrafficActionPassthrough
	default:
		return ReviewTrafficActionSafePage
	}
}

func (a ReviewTrafficAction) Valid() bool {
	switch a {
	case ReviewTrafficActionSafePage, ReviewTrafficActionBlock, ReviewTrafficActionPassthrough:
		return true
	default:
		return false
	}
}
