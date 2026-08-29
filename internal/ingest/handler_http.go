package ingest

import (
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingest/httpingress"
)

type h2ConnState = httpingress.H2ConnState

var (
	errIncompleteRequest = httpingress.ErrIncomplete
	errInvalidRequest    = httpingress.ErrInvalid
	errPayloadTooLarge   = httpingress.ErrPayloadTooLarge
)

func init() {
	httpingress.RegisterFilterHooks()
	httpingress.SetIngressPathValidFn(http1IngressPathValid)
	httpingress.SetHeaderOrderPathRecordsFn(http1HeaderOrderPathRecords)
	httpingress.SetAssignConnTimingHeadersFn(func(req *httpingress.Request, key, val []byte) {
		http1AssignConnTimingHeaders(req, key, val)
	})
	httpingress.SetHeaderOrderPolicy(httpingress.HeaderOrderPolicy{
		ChromeNotChromium: uaClaimsChromeNotChromium,
		InAppWebView:      uaMatchesInAppWebView,
		SecFetchAllBits:   wireSecFetchAllBits,
	})
}

func http1IngressPathValid(method, path []byte) bool {
	if len(method) == 4 && method[0] == 'P' && method[1] == 'O' && method[2] == 'S' && method[3] == 'T' {
		return httpingress.PathHasPrefix(path, "/track") || httpingress.PathHasPrefix(path, "/openrtb/bid") || httpingress.PathHasPrefix(path, "/tg/bid")
	}
	if len(method) == 7 && method[0] == 'O' && method[1] == 'P' && method[2] == 'T' &&
		method[3] == 'I' && method[4] == 'O' && method[5] == 'N' && method[6] == 'S' {
		return httpingress.BytesEqual(path, "/track")
	}
	if len(method) == 3 && method[0] == 'G' && method[1] == 'E' && method[2] == 'T' {
		return httpingress.BytesEqual(path, "/health") ||
			httpingress.BytesEqual(path, "/healthz") ||
			httpingress.BytesEqual(path, "/ready") ||
			httpingress.BytesEqual(path, "/readyz") ||
			httpingress.BytesEqual(path, "/metrics") ||
			isTrackPixelPath(path) ||
			httpingress.PathHasPrefix(path, safePageStubPathPrefix) ||
			httpingress.PathHasPrefix(path, "/click") ||
			httpingress.PathHasPrefix(path, telegramPathClick) ||
			httpingress.PathHasPrefix(path, telegramPathImpression)
	}
	return false
}

func http1HeaderOrderPathRecords(method, path []byte) bool {
	if len(method) == 4 && method[0] == 'P' && method[1] == 'O' && method[2] == 'S' && method[3] == 'T' {
		return httpingress.BytesEqual(path, "/track")
	}
	if len(method) == 3 && method[0] == 'G' && method[1] == 'E' && method[2] == 'T' {
		return httpingress.PathHasPrefix(path, "/click") ||
			httpingress.PathHasPrefix(path, telegramPathClick) ||
			httpingress.PathHasPrefix(path, telegramPathImpression)
	}
	return false
}

func parseHTTP1(data []byte, maxBody int64, scratchPtr *[]byte) (int, Request, error) {
	return httpingress.ParseHTTP1(data, maxBody, scratchPtr)
}

func parseHTTP1ChunkedBody(data []byte, off int, maxBody int64, scratchPtr *[]byte) (int, []byte, int, error) {
	return httpingress.ParseHTTP1ChunkedBody(data, off, maxBody, scratchPtr)
}

func parseH2Ingress(buf []byte, st *h2ConnState, maxBody int64) (int, Request, uint32, []byte, error) {
	return httpingress.ParseH2Ingress(buf, st, maxBody)
}

func isH2ClientPreface(buf []byte) bool {
	return httpingress.IsH2ClientPreface(buf)
}

func h3ParseRequestFrames(buf []byte, maxBody int64) (int, Request, error) {
	return httpingress.H3ParseRequestFrames(buf, maxBody)
}

func resetChunkScratch(scratchPtr *[]byte) {
	httpingress.ResetChunkScratch(scratchPtr)
}

func http1HeadersComplete(data []byte) bool {
	return httpingress.HeadersComplete(data)
}

func http1AssignHeader(req *Request, key, val []byte, hFlags *uint8, clValue *int) error {
	return httpingress.AssignHeader(req, key, val, hFlags, clValue)
}

func http1AssignWireMetadataHeaders(req *Request, key, val []byte) {
	httpingress.AssignWireMetadataHeaders(req, key, val)
}

func http1KeyMatchFold(key []byte, lit string) bool {
	return httpingress.KeyMatchFold(key, lit)
}

func edgeHTTP1Disposition(wire []byte, maxBody int64) ingressDisposition {
	return httpingress.EdgeHTTP1Disposition(wire, maxBody)
}

func gnetHTTP1Disposition(wire []byte, maxBody int64) ingressDisposition {
	return httpingress.GnetHTTP1Disposition(wire, maxBody)
}

func http1IngressCanonical(wire []byte, maxBody int64) (ingressDisposition, ingressDisposition, bool) {
	return httpingress.HTTP1IngressCanonical(wire, maxBody)
}

func buildNginxIngressCorpus() []ingressCorpusCase {
	return httpingress.BuildNginxIngressCorpus()
}

func http1FaultMalformedCases() []http1FaultCase {
	return httpingress.HTTP1FaultMalformedCases()
}

func fraudHTTP1Cases2026() []fraudHTTP1Case {
	return httpingress.FraudHTTP1Cases2026()
}

func dispositionFromHTTP1Parse(n int, req Request, err error) ingressDisposition {
	return httpingress.DispositionFromHTTP1Parse(n, req, err)
}

func randomWireGarbage(n int) []byte {
	return httpingress.RandomWireGarbage(n)
}

func copyHTTP1HeaderOrderToEvent(evt *domain.Event, req *Request) {
	if evt == nil || req == nil {
		return
	}
	evt.HTTP1HeaderOrderCount = req.HTTP1HeaderOrderCount
	if req.HTTP1HeaderOrderCount == 0 {
		return
	}
	n := req.HTTP1HeaderOrderCount
	if n > httpingress.HeaderOrderMax {
		n = httpingress.HeaderOrderMax
	}
	copy(evt.HTTP1HeaderOrder[:n], req.HTTP1HeaderOrder[:n])
}

type ingressDisposition = httpingress.Disposition

type ingressCorpusCase = httpingress.CorpusCase

type IngressVerdict = httpingress.IngressVerdict

type http1FaultCase = httpingress.HTTP1FaultCase

type fraudHTTP1Case = httpingress.FraudHTTP1Case

var nginxTrackCorpus = httpingress.NginxTrackCorpus

const http1MaxBufferedOverhead = httpingress.MaxBufferedOverhead

func bytesEqual(b []byte, s string) bool {
	return httpingress.BytesEqual(b, s)
}

func httpPathHasPrefix(path []byte, prefix string) bool {
	return httpingress.PathHasPrefix(path, prefix)
}

func fillWireMetadataFromRequest(evt *domain.Event, req *Request) {
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

func fillH2WireFromRequest(evt *domain.Event, req *Request) {
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
