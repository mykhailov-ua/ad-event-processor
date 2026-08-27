package postback

import "strings"

func ResolveEventID(pb *PostbackPayload) string {
	if pb == nil {
		return ""
	}
	if id := strings.TrimSpace(pb.EventID); id != "" {
		return id
	}
	if id := strings.TrimSpace(pb.TxID); id != "" {
		return id
	}
	return strings.TrimSpace(pb.ClickID)
}
