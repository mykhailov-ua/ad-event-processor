package gnet

// PinHTTP1RequestInPlace copies header/body slices into ConnContext.OffloadHTTPPin so Tier B
// can run after gnet discards the peek frame. Slices in req alias the pin buffer on return.
func PinHTTP1RequestInPlace(ctx *ConnContext, req *Request) {
	if ctx == nil || req == nil {
		return
	}
	ctx.OffloadHTTPPin = ctx.OffloadHTTPPin[:0]
	pin := func(b []byte) []byte {
		if len(b) == 0 {
			return nil
		}
		off := len(ctx.OffloadHTTPPin)
		ctx.OffloadHTTPPin = append(ctx.OffloadHTTPPin, b...)
		return ctx.OffloadHTTPPin[off : off+len(b)]
	}
	req.Method = pin(req.Method)
	req.Path = pin(req.Path)
	req.ContentType = pin(req.ContentType)
	req.ClientIP = pin(req.ClientIP)
	req.RealIP = pin(req.RealIP)
	req.UserAgent = pin(req.UserAgent)
	req.Accept = pin(req.Accept)
	req.AcceptEncoding = pin(req.AcceptEncoding)
	req.TLSHash = pin(req.TLSHash)
	req.TLSJA3 = pin(req.TLSJA3)
	req.TLSJA4 = pin(req.TLSJA4)
	req.SecCHUA = pin(req.SecCHUA)
	req.SecCHUAPlatform = pin(req.SecCHUAPlatform)
	req.SecCHUAMobile = pin(req.SecCHUAMobile)
	req.SecFetchSite = pin(req.SecFetchSite)
	req.SecFetchMode = pin(req.SecFetchMode)
	req.SecFetchDest = pin(req.SecFetchDest)
	req.TLSALPN = pin(req.TLSALPN)
	req.AcceptLang = pin(req.AcceptLang)
	req.Body = pin(req.Body)
	req.Origin = pin(req.Origin)
	req.Host = pin(req.Host)
	req.Cookie = pin(req.Cookie)
}
