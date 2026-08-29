package track

import (
	"bytes"
	_ "embed"
	"strings"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
)

type SafePageDelivery uint8

const (
	SafePageDeliveryInPlace SafePageDelivery = iota
	SafePageDeliveryRedirect
)

type SafePageAction uint8

const (
	SafePageActionNone SafePageAction = iota
	SafePageActionInPlace
	SafePageActionRedirect
)

func SafePageEligibleReject(kind filter.FilterRejectKind) bool {
	switch kind {
	case filter.FilterRejectFraud, filter.FilterRejectPlacementBlocked:
		return true
	default:
		return false
	}
}

func ResolveSafePageLanding(registry domain.CampaignRegistry, campaignID uuid.UUID) (string, bool) {
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

func ResolveSafePageAction(
	registry domain.CampaignRegistry,
	campaignID uuid.UUID,
	outcome Outcome,
	forceSafe bool,
) (SafePageAction, string) {
	return ResolveSafePageActionDelivery(registry, campaignID, outcome, forceSafe, SafePageDeliveryInPlace)
}

func ResolveSafePageActionDelivery(
	registry domain.CampaignRegistry,
	campaignID uuid.UUID,
	outcome Outcome,
	forceSafe bool,
	delivery SafePageDelivery,
) (SafePageAction, string) {
	url, ok := ResolveSafePageLanding(registry, campaignID)
	if !ok {
		return SafePageActionNone, ""
	}
	eligible := forceSafe ||
		outcome.Status == StatusFraudAccepted ||
		(outcome.Status == StatusRejected && SafePageEligibleReject(outcome.RejectKind))
	if !eligible {
		return SafePageActionNone, ""
	}
	metrics.SafePageRedirectTotal.Inc()
	if delivery == SafePageDeliveryRedirect {
		return SafePageActionRedirect, url
	}
	return SafePageActionInPlace, url
}

const SafePageStubPathPrefix = "/safe_page_stub"

func AppendSafePageStubPath(dst []byte, campaignID uuid.UUID) []byte {
	dst = append(dst, SafePageStubPathPrefix...)
	dst = append(dst, "?campaign_id="...)
	return append(dst, campaignID.String()...)
}

func ParseSafePageStubCampaignID(path []byte) (uuid.UUID, bool) {
	if !bytes.HasPrefix(path, []byte(SafePageStubPathPrefix)) {
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
	id, err := uuid.Parse(filter.UnsafeString(path[start:end]))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

//go:embed safe_page_hydrator.js
var safePageHydratorJS []byte

var (
	SafePageStubHTMLHead = []byte("<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>Loading</title></head><body><main><iframe src=\"")
	SafePageStubHTMLMid  = []byte("\" title=\"content\" style=\"border:0;width:100%;height:100vh\"></iframe></main><script>")
	SafePageStubHTMLTail = []byte("</script></body></html>")
)

func AppendSafePageStubBody(dst []byte, safeURL []byte) []byte {
	dst = append(dst, SafePageStubHTMLHead...)
	dst = append(dst, safeURL...)
	dst = append(dst, SafePageStubHTMLMid...)
	dst = append(dst, safePageHydratorJS...)
	dst = append(dst, SafePageStubHTMLTail...)
	return dst
}

var (
	SafePageStubHTTPPrefix = []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\nConnection: keep-alive\r\nContent-Length: ")
	SafePageStubHTTPMiddle = []byte("\r\n\r\n")
)

func SafePageStubWireLen(urlBytes []byte) int {
	bodyLen := len(SafePageStubHTMLHead) + len(urlBytes) + len(SafePageStubHTMLMid) + len(safePageHydratorJS) + len(SafePageStubHTMLTail)
	return len(SafePageStubHTTPPrefix) + BodyLenDigits(bodyLen) + len(SafePageStubHTTPMiddle) + bodyLen
}

func BuildSafePageStubWire(dst []byte, urlBytes []byte) []byte {
	body := AppendSafePageStubBody(nil, urlBytes)
	dst = append(dst, SafePageStubHTTPPrefix...)
	dst = appendInt(dst, int64(len(body)))
	dst = append(dst, SafePageStubHTTPMiddle...)
	dst = append(dst, body...)
	return dst
}

func BodyLenDigits(n int) int {
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

func SafePageURLAttrBytes(url string) ([]byte, bool) {
	return safePageURLAttrBytes(url)
}

func ClickHasTTCFraudSignal(evt *domain.Event, missingImpTS, lowTTC string) bool {
	if evt == nil || evt.FraudReason == "" {
		return false
	}
	reason := evt.FraudReason
	return strings.Contains(reason, missingImpTS) || strings.Contains(reason, lowTTC)
}
