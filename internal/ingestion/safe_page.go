package ingestion

import (
	"bytes"
	_ "embed"
	"net/http"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
)

//go:embed safe_page_hydrator.js
var safePageHydratorJS []byte

type safePageDelivery uint8

const (
	safePageDeliveryInPlace safePageDelivery = iota
	safePageDeliveryRedirect
)

type safePageAction uint8

const (
	safePageActionNone safePageAction = iota
	safePageActionInPlace
	safePageActionRedirect
)

func safePageEligibleReject(kind filterRejectKind) bool {
	switch kind {
	case filterRejectFraud, filterRejectPlacementBlocked:
		return true
	default:
		return false
	}
}

func resolveSafePageLanding(registry domain.CampaignRegistry, campaignID uuid.UUID) (string, bool) {
	if registry == nil {
		return "", false
	}
	camp, ok := registry.GetCampaign(campaignID)
	if !ok || camp == nil || !camp.SafePageEnabled {
		return "", false
	}
	if camp.SafePageURL == "" {
		return "", false
	}
	return camp.SafePageURL, true
}

func resolveSafePageAction(
	registry domain.CampaignRegistry,
	campaignID uuid.UUID,
	outcome trackOutcome,
	forceSafe bool,
) (safePageAction, string) {
	return resolveSafePageActionDelivery(registry, campaignID, outcome, forceSafe, safePageDeliveryInPlace)
}

func resolveSafePageActionDelivery(
	registry domain.CampaignRegistry,
	campaignID uuid.UUID,
	outcome trackOutcome,
	forceSafe bool,
	delivery safePageDelivery,
) (safePageAction, string) {
	url, ok := resolveSafePageLanding(registry, campaignID)
	if !ok {
		return safePageActionNone, ""
	}
	eligible := forceSafe ||
		outcome.Status == trackStatusFraudAccepted ||
		(outcome.Status == trackStatusRejected && safePageEligibleReject(outcome.RejectKind))
	if !eligible {
		return safePageActionNone, ""
	}
	metrics.SafePageRedirectTotal.Inc()
	if delivery == safePageDeliveryRedirect {
		return safePageActionRedirect, url
	}
	return safePageActionInPlace, url
}

const safePageStubPathPrefix = "/safe_page_stub"

var (
	safePageStubHTMLHead = []byte("<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>BidShard</title></head><body><main><iframe src=\"")
	safePageStubHTMLMid  = []byte("\" title=\"content\" style=\"border:0;width:100%;height:100vh\"></iframe></main><script>")
	safePageStubHTMLTail = []byte("</script></body></html>")
)

func parseSafePageStubCampaignID(path []byte) (uuid.UUID, bool) {
	if !bytes.HasPrefix(path, []byte(safePageStubPathPrefix)) {
		return uuid.Nil, false
	}
	key := []byte("campaign_id=")
	idx := bytes.Index(path, key)
	if idx < 0 {
		return uuid.Nil, false
	}
	start := idx + len(key)
	end := start
	for end < len(path) && path[end] != '&' {
		end++
	}
	if end-start != 36 {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(unsafeString(path[start:end]))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func safePageURLAttrBytes(url string) ([]byte, bool) {
	if url == "" || len(url) > 2048 {
		return nil, false
	}
	u := UnsafeBytes(url)
	if !bytes.HasPrefix(u, []byte("https://")) && !bytes.HasPrefix(u, []byte("http://")) {
		return nil, false
	}
	for _, b := range u {
		if b < 0x20 || b == '"' || b == '<' || b == '>' {
			return nil, false
		}
	}
	return u, true
}

func appendSafePageStubBody(dst []byte, safeURL []byte) []byte {
	dst = append(dst, safePageStubHTMLHead...)
	dst = append(dst, safeURL...)
	dst = append(dst, safePageStubHTMLMid...)
	dst = append(dst, safePageHydratorJS...)
	dst = append(dst, safePageStubHTMLTail...)
	return dst
}

func writeSafePageStubResponse(h *AdsPacketHandler, c gnet.Conn, ctx *connContext, campaignID uuid.UUID) {
	url, ok := resolveSafePageLanding(h.registry, campaignID)
	if !ok {
		h.write(c, respClickNoLanding, ctx)
		return
	}
	urlBytes, ok := safePageURLAttrBytes(url)
	if !ok {
		h.write(c, respClickBadLanding, ctx)
		return
	}
	bodyLen := len(safePageStubHTMLHead) + len(urlBytes) + len(safePageStubHTMLMid) + len(safePageHydratorJS) + len(safePageStubHTMLTail)
	total := len(safePageStubHTTPPrefix) + bodyLenDigits(bodyLen) + len(safePageStubHTTPMiddle) + bodyLen
	buf := ctx.bufSlice
	if cap(buf) < total {
		buf = make([]byte, total, total+64)
		ctx.bufSlice = buf
	} else {
		buf = buf[:total]
	}
	off := copy(buf, safePageStubHTTPPrefix)
	off += appendInt(buf[off:], int64(bodyLen))
	off += copy(buf[off:], safePageStubHTTPMiddle)
	body := appendSafePageStubBody(nil, urlBytes)
	copy(buf[off:], body)
	h.write(c, buf[:off+len(body)], ctx)
}

func (h *AdsPacketHandler) reactSafePageStub(req parsedHTTPRequest, c gnet.Conn, ctx *connContext) gnet.Action {
	startMono := monotonicNano()
	campaignID, ok := parseSafePageStubCampaignID(req.Path)
	if !ok {
		h.write(c, respClickBadRequest, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}
	writeSafePageStubResponse(h, c, ctx, campaignID)
	h.recordMetrics(startMono, http.StatusOK)
	return gnet.None
}

func bodyLenDigits(n int) int {
	if n <= 0 {
		return 1
	}
	d := 0
	for n > 0 {
		d++
		n /= 10
	}
	return d
}

var (
	safePageStubHTTPPrefix = []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\nConnection: keep-alive\r\nContent-Length: ")
	safePageStubHTTPMiddle = []byte("\r\n\r\n")
)
