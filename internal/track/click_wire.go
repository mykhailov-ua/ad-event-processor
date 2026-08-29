package track

import (
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
)

func ConnTypePolicyBlocks(policy domain.ConnTypePolicy, match bool, connType uint8) bool {
	switch policy {
	case domain.ConnTypeMobileOnly:
		if !match {
			return true
		}
		return connType&filter.ProxyVPNConnMobile == 0
	case domain.ConnTypeResidentialOnly:
		if !match {
			return true
		}
		if connType&(filter.ProxyVPNConnVPN|filter.ProxyVPNConnHosting|filter.ProxyVPNConnMobile) != 0 {
			return true
		}
		return connType&filter.ProxyVPNConnISP == 0
	default:
		return match && filter.ProxyVPNConnTypeBlocks(connType)
	}
}

const (
	ClickRedirectHdrPrefix = "HTTP/1.1 302 Found\r\nLocation: "
	ClickRedirectHdrSuffix = "\r\nReferrer-Policy: no-referrer\r\nCache-Control: no-store\r\nContent-Length: 0\r\nConnection: keep-alive\r\n\r\n"
	RedirectWireMinCap     = 512
)

func BuildClickRedirectWire(dst []byte, location []byte) []byte {
	total := len(ClickRedirectHdrPrefix) + len(location) + len(ClickRedirectHdrSuffix)
	if cap(dst) < total {
		if cap(dst) < RedirectWireMinCap {
			dst = make([]byte, total, RedirectWireMinCap)
		} else {
			dst = make([]byte, total)
		}
	} else {
		dst = dst[:total]
	}
	off := copy(dst, ClickRedirectHdrPrefix)
	off += copy(dst[off:], location)
	copy(dst[off:], ClickRedirectHdrSuffix)
	return dst
}
