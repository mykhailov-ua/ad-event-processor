package adminapi

import "strings"

// classifyTrafficChannel maps attribution query params to a stable channel taxonomy.
func classifyTrafficChannel(sub1, sub2, gclid, fbclid, ttclid string) string {
	if strings.TrimSpace(gclid) != "" {
		return "paid_search"
	}
	if strings.TrimSpace(fbclid) != "" || strings.TrimSpace(ttclid) != "" {
		return "paid_social"
	}
	sub1 = strings.ToLower(strings.TrimSpace(sub1))
	sub2 = strings.ToLower(strings.TrimSpace(sub2))
	switch {
	case sub1 == "organic" || sub2 == "organic":
		return "organic"
	case sub1 == "email" || sub2 == "email":
		return "email"
	case sub1 == "affiliate" || sub2 == "affiliate":
		return "affiliate"
	case sub1 != "" || sub2 != "":
		return "custom"
	default:
		return "direct"
	}
}
