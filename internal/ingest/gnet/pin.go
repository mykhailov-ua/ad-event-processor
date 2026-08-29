package gnet

func PinParsedHTTPRequest(ctx *ConnContext, req Request) Request {
	ctx.OffloadHTTPPin = ctx.OffloadHTTPPin[:0]
	pin := func(b []byte) []byte {
		if len(b) == 0 {
			return nil
		}
		off := len(ctx.OffloadHTTPPin)
		ctx.OffloadHTTPPin = append(ctx.OffloadHTTPPin, b...)
		return ctx.OffloadHTTPPin[off : off+len(b)]
	}
	return Request{
		Method:                pin(req.Method),
		Path:                  pin(req.Path),
		ContentType:           pin(req.ContentType),
		ClientIP:              pin(req.ClientIP),
		UserAgent:             pin(req.UserAgent),
		Accept:                pin(req.Accept),
		AcceptEncoding:        pin(req.AcceptEncoding),
		TLSHash:               pin(req.TLSHash),
		TLSJA3:                pin(req.TLSJA3),
		TLSJA4:                pin(req.TLSJA4),
		SecCHUA:               pin(req.SecCHUA),
		SecCHUAPlatform:       pin(req.SecCHUAPlatform),
		SecCHUAMobile:         pin(req.SecCHUAMobile),
		SecFetchSite:          pin(req.SecFetchSite),
		SecFetchMode:          pin(req.SecFetchMode),
		SecFetchDest:          pin(req.SecFetchDest),
		TLSALPN:               pin(req.TLSALPN),
		SecFetchPresent:       req.SecFetchPresent,
		H2WireFlags:           req.H2WireFlags,
		H2SettingsCRC:         req.H2SettingsCRC,
		H2EnablePush:          req.H2EnablePush,
		H2InitialWindow:       req.H2InitialWindow,
		H2WindowUpdateInc:     req.H2WindowUpdateInc,
		H2PseudoOrder:         req.H2PseudoOrder,
		H2PseudoOrderCount:    req.H2PseudoOrderCount,
		HTTP1HeaderOrder:      req.HTTP1HeaderOrder,
		HTTP1HeaderOrderCount: req.HTTP1HeaderOrderCount,
		AcceptLang:            pin(req.AcceptLang),
		Body:                  pin(req.Body),
		Origin:                pin(req.Origin),
		Host:                  pin(req.Host),
		Cookie:                pin(req.Cookie),
		ContentLength:         req.ContentLength,
		HasContentLength:      req.HasContentLength,
		ForceSafe:             req.ForceSafe,
		TCPMSS:                req.TCPMSS,
		TCPMSSSet:             req.TCPMSSSet,
		TCPTTL:                req.TCPTTL,
		TCPTTLSet:             req.TCPTTLSet,
		TCPWindow:             req.TCPWindow,
		TCPWindowSet:          req.TCPWindowSet,
		TCPSig:                req.TCPSig,
		TCPSigSet:             req.TCPSigSet,
		RTTSynMS:              req.RTTSynMS,
		TTFBAppMS:             req.TTFBAppMS,
		ConnTimingSet:         req.ConnTimingSet,
	}
}
