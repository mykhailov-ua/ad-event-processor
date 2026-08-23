package ingestion

func pinParsedHTTPRequest(ctx *connContext, req parsedHTTPRequest) parsedHTTPRequest {
	ctx.offloadHTTPPin = ctx.offloadHTTPPin[:0]
	pin := func(b []byte) []byte {
		if len(b) == 0 {
			return nil
		}
		off := len(ctx.offloadHTTPPin)
		ctx.offloadHTTPPin = append(ctx.offloadHTTPPin, b...)
		return ctx.offloadHTTPPin[off : off+len(b)]
	}
	return parsedHTTPRequest{
		Method:           pin(req.Method),
		Path:             pin(req.Path),
		ContentType:      pin(req.ContentType),
		ClientIP:         pin(req.ClientIP),
		UserAgent:        pin(req.UserAgent),
		Accept:           pin(req.Accept),
		AcceptEncoding:   pin(req.AcceptEncoding),
		TLSHash:          pin(req.TLSHash),
		TLSJA3:           pin(req.TLSJA3),
		TLSJA4:           pin(req.TLSJA4),
		SecCHUA:          pin(req.SecCHUA),
		AcceptLang:       pin(req.AcceptLang),
		Body:             pin(req.Body),
		ContentLength:    req.ContentLength,
		HasContentLength: req.HasContentLength,
		ForceSafe:        req.ForceSafe,
		TCPMSS:           req.TCPMSS,
		TCPMSSSet:        req.TCPMSSSet,
		TCPTTL:           req.TCPTTL,
		TCPTTLSet:        req.TCPTTLSet,
		TCPWindow:        req.TCPWindow,
		TCPWindowSet:     req.TCPWindowSet,
	}
}
