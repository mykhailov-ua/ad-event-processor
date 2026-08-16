package postback

import (
	"net/url"
	"strings"

	"github.com/google/uuid"
)

const defaultEventSourceHost = "tracking.invalid"

// synthesizeEventSourceURL builds the tracking-domain click URL for CAPI (not upstream proxy URL).
func synthesizeEventSourceURL(pb PostbackPayload, trackingHost string) string {
	if pb.CampaignID == uuid.Nil || pb.ClickID == "" {
		return ""
	}
	q := url.Values{}
	q.Set("campaign_id", pb.CampaignID.String())
	q.Set("type", "click")
	q.Set("click_id", pb.ClickID)
	if pb.GCLID != "" {
		q.Set("gclid", pb.GCLID)
	}
	if pb.FBCLID != "" {
		q.Set("fbclid", pb.FBCLID)
	}
	if pb.TTCLID != "" {
		q.Set("ttclid", pb.TTCLID)
	}
	subs := pb.SubIDs()
	for i := range subs {
		if subs[i] == "" {
			continue
		}
		q.Set(subIDJSONKey(i+1, false), subs[i])
	}
	host := strings.TrimSpace(trackingHost)
	if host == "" {
		host = defaultEventSourceHost
	}
	host = strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://")
	return "https://" + host + "/click?" + q.Encode()
}
