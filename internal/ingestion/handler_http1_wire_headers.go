package ingestion

import "ad-event-processor/internal/domain"

func http1KeyMatchFold(key []byte, lit string) bool {
	if len(key) != len(lit) {
		return false
	}
	for i := 0; i < len(lit); i++ {
		if httpFold[key[i]] != lit[i] {
			return false
		}
	}
	return true
}

func http1AssignWireMetadataHeaders(req *parsedHTTPRequest, key, val []byte) {
	switch {
	case http1KeyMatchFold(key, "sec-fetch-site"):
		req.SecFetchSite = val
		req.SecFetchPresent |= wireSecFetchSiteBit
	case http1KeyMatchFold(key, "sec-fetch-mode"):
		req.SecFetchMode = val
		req.SecFetchPresent |= wireSecFetchModeBit
	case http1KeyMatchFold(key, "sec-fetch-dest"):
		req.SecFetchDest = val
		req.SecFetchPresent |= wireSecFetchDestBit
	case http1KeyMatchFold(key, "sec-ch-ua-platform"):
		req.SecCHUAPlatform = val
	case http1KeyMatchFold(key, "sec-ch-ua-mobile"):
		req.SecCHUAMobile = val
	case http1KeyMatchFold(key, "x-tls-alpn"):
		req.TLSALPN = val
	}
	http1AssignConnTimingHeaders(req, key, val)
}

func fillWireMetadataFromRequest(evt *domain.Event, req *parsedHTTPRequest) {
	if evt == nil || req == nil {
		return
	}
	evt.SecCHUAPlatform = unsafeString(req.SecCHUAPlatform)
	evt.TLSALPN = unsafeString(req.TLSALPN)
	evt.SecFetchPresent = req.SecFetchPresent
	evt.SecFetchSite = classifySecFetchSite(req.SecFetchSite)
	evt.SecFetchMode = classifySecFetchMode(req.SecFetchMode)
	evt.SecFetchDest = classifySecFetchDest(req.SecFetchDest)
	evt.SecCHUAMobile = classifySecCHUAMobile(req.SecCHUAMobile)
	if req.AcceptEncoding != nil {
		evt.AcceptEncodingSet = 1
		evt.AcceptEncodingFlags = classifyAcceptEncoding(req.AcceptEncoding)
	}
	copyHTTP1HeaderOrderToEvent(evt, req)
	fillH2WireFromRequest(evt, req)
}

func fillH2WireFromRequest(evt *domain.Event, req *parsedHTTPRequest) {
	if evt == nil || req == nil {
		return
	}
	evt.H2WireFlags = req.H2WireFlags
	evt.H2SettingsCRC = req.H2SettingsCRC
	evt.H2EnablePush = req.H2EnablePush
	evt.H2InitialWindow = req.H2InitialWindow
	evt.H2WindowUpdateInc = req.H2WindowUpdateInc
	evt.H2PseudoOrder = req.H2PseudoOrder
	evt.H2PseudoOrderCount = req.H2PseudoOrderCount
}

func fillIngressH2(evt *domain.Event, protoH2 bool) {
	if evt == nil {
		return
	}
	if protoH2 {
		evt.IngressH2 = 1
	} else {
		evt.IngressH2 = 0
	}
}
