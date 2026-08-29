package track

import (
	"strconv"

	"ad-event-processor/pkg/branding"
)

var SafeViewCIDRBody = []byte(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Loading</title></head><body><main><p>Please wait&hellip;</p></main></body></html>`)

var (
	RespClickSafeViewCIDR         = buildSafeViewResponse("l1")
	RespClickSafeViewIPv4Rotation = buildSafeViewResponse("l1v4")
	RespClickSafeViewIPv6Rotation = buildSafeViewResponse("l1v6")
	RespClickSafeViewProxyVPN     = buildSafeViewResponse("l15")
	RespClickSafeViewTLS          = buildSafeViewResponse("tls")
	RespClickSafeViewModerator    = buildSafeViewResponse("moderator")
)

func buildSafeViewResponse(tag string) []byte {
	head := "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\n" + branding.HTTPSafeViewHeader + ": " + tag + "\r\nConnection: keep-alive\r\nContent-Length: "
	out := make([]byte, 0, len(head)+5+4+len(SafeViewCIDRBody))
	out = append(out, head...)
	out = strconv.AppendInt(out, int64(len(SafeViewCIDRBody)), 10)
	out = append(out, "\r\n\r\n"...)
	out = append(out, SafeViewCIDRBody...)
	return out
}
