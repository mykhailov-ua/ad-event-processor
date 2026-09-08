package postback

import "strings"

func capiPostbackProvider(provider string) bool {
	switch strings.TrimSpace(strings.ToLower(provider)) {
	case "facebook", "google", "tiktok", "microsoft_ads":
		return true
	default:
		return false
	}
}
