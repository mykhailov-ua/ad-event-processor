package ingestion

import (
	"github.com/google/uuid"
)

func appendTelegramClickLink(dst []byte, baseURL string, campaignID, clickID uuid.UUID, widgetID []byte) []byte {
	dst = append(dst, baseURL...)
	sep := byte('?')
	for i := 0; i < len(dst); i++ {
		if dst[i] == '?' {
			sep = '&'
			break
		}
	}
	dst = append(dst, sep)
	dst = append(dst, "campaign_id="...)
	dst = appendUUIDStr(dst, campaignID)
	dst = append(dst, "&click_id="...)
	dst = appendUUIDStr(dst, clickID)
	if len(widgetID) > 0 {
		dst = append(dst, "&widget_id="...)
		dst = append(dst, widgetID...)
	}
	return dst
}
