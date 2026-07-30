package perimeter

import "strings"

type ASNWhitelist struct {
	cdn    map[string]struct{}
	mobile map[string]struct{}
}

func NewASNWhitelist(cdnRaw, mobileRaw string) *ASNWhitelist {
	return &ASNWhitelist{
		cdn:    parseASNSet(cdnRaw),
		mobile: parseASNSet(mobileRaw),
	}
}

func parseASNSet(raw string) map[string]struct{} {
	if raw == "" {
		return nil
	}
	out := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		asn := strings.TrimSpace(part)
		if asn != "" {
			out[asn] = struct{}{}
		}
	}
	return out
}

func (w *ASNWhitelist) IsWhitelisted(asn string) bool {
	if w == nil || asn == "" {
		return false
	}
	asn = strings.TrimSpace(asn)
	if _, ok := w.cdn[asn]; ok {
		return true
	}
	if _, ok := w.mobile[asn]; ok {
		return true
	}
	return false
}
