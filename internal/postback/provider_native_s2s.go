package postback

import "strings"

func ProviderRequiresToken(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "webhook", "taboola", "outbrain":
		return false
	default:
		return true
	}
}

func resolveOutboundEventName(urlTemplate, eventType, fallback string) string {
	t := strings.TrimSpace(urlTemplate)
	if t != "" && !strings.HasPrefix(t, "http") && !strings.Contains(t, "|") && !strings.HasPrefix(t, "{") {
		return t
	}
	if e := strings.TrimSpace(eventType); e != "" {
		return e
	}
	return fallback
}

func resolveTaboolaClickID(payload *PostbackPayload) string {
	if payload == nil {
		return ""
	}
	if v := strings.TrimSpace(payload.TBLCI); v != "" {
		return v
	}
	return strings.TrimSpace(payload.ClickID)
}

func resolveOutbrainClickID(payload *PostbackPayload) string {
	if payload == nil {
		return ""
	}
	if v := strings.TrimSpace(payload.OBClickID); v != "" {
		return v
	}
	return strings.TrimSpace(payload.ClickID)
}

func resolveMicrosoftClickID(payload *PostbackPayload) string {
	if payload == nil {
		return ""
	}
	if v := strings.TrimSpace(payload.MSCLKID); v != "" {
		return v
	}
	return strings.TrimSpace(payload.ClickID)
}
