package campaign

import (
	"net/http"
	"strings"
)

func CampaignRevision(updatedAt string) string {
	return campaignRevision(updatedAt)
}

func campaignRevision(updatedAt string) string {
	return strings.TrimSpace(updatedAt)
}

func resolveExpectedRevision(r *http.Request, req *PatchCampaignRequest) {
	if req.ExpectedRevision != nil && strings.TrimSpace(*req.ExpectedRevision) != "" {
		return
	}
	ifMatch := strings.TrimSpace(r.Header.Get("If-Match"))
	if ifMatch != "" {
		req.ExpectedRevision = &ifMatch
	}
}
