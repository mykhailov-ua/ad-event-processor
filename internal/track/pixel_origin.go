package track

import (
	"fmt"
	"net/url"
	"strings"
)

const TrackPixelFirstPartyPath = "/_aed/track.js"

type BrowserPixelBundle struct {
	ScriptURL  string
	TrackURL   string
	Snippet    string
	FirstParty bool
}

func ResolveBrowserPixel(landerPublicBase, trackingBase, campaignID string) BrowserPixelBundle {
	lander := normalizePublicBase(landerPublicBase)
	tracker := normalizePublicBase(trackingBase)
	if tracker == "" {
		tracker = lander
	}
	trackURL := strings.TrimRight(tracker, "/") + "/track"

	scriptURL := strings.TrimRight(tracker, "/") + TrackPixelPath
	firstParty := false
	if lander != "" {
		scriptURL = strings.TrimRight(lander, "/") + TrackPixelFirstPartyPath
		firstParty = true
	}

	campaignID = strings.TrimSpace(campaignID)
	snippet := BuildBrowserPixelSnippet(scriptURL, trackURL, campaignID)
	return BrowserPixelBundle{
		ScriptURL:  scriptURL,
		TrackURL:   trackURL,
		Snippet:    snippet,
		FirstParty: firstParty,
	}
}

func BuildBrowserPixelSnippet(scriptURL, trackURL, campaignID string) string {
	scriptURL = strings.TrimSpace(scriptURL)
	trackURL = strings.TrimSpace(trackURL)
	campaignID = strings.TrimSpace(campaignID)
	if scriptURL == "" || trackURL == "" || campaignID == "" {
		return ""
	}
	return fmt.Sprintf(`<script src="%s"></script>
<script>
  const conversionEventId = crypto.randomUUID();
  trackEvent({
    campaignId: '%s',
    type: 'conversion',
    endpoint: '%s',
    eventId: conversionEventId,
  });
</script>`, scriptURL, campaignID, trackURL)
}

func normalizePublicBase(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return strings.TrimRight(raw, "/")
	}
	scheme := u.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + u.Host
}
