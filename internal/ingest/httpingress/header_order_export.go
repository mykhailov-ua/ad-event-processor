package httpingress

const (
	HTTP1HdrNone            = http1HdrNone
	HTTP1HdrHost            = http1HdrHost
	HTTP1HdrConnection      = http1HdrConnection
	HTTP1HdrSecCHUA         = http1HdrSecCHUA
	HTTP1HdrSecCHUAMobile   = http1HdrSecCHUAMobile
	HTTP1HdrSecCHUAPlatform = http1HdrSecCHUAPlatform
	HTTP1HdrUpgradeInsecure = http1HdrUpgradeInsecure
	HTTP1HdrUserAgent       = http1HdrUserAgent
	HTTP1HdrAccept          = http1HdrAccept
	HTTP1HdrSecFetchSite    = http1HdrSecFetchSite
	HTTP1HdrSecFetchMode    = http1HdrSecFetchMode
	HTTP1HdrSecFetchDest    = http1HdrSecFetchDest
	HTTP1HdrAcceptEncoding  = http1HdrAcceptEncoding
	HTTP1HdrAcceptLanguage  = http1HdrAcceptLanguage
	HTTP1HdrContentType     = http1HdrContentType
)

func ClassifyHTTP1HeaderOrderToken(key []byte) uint8 {
	return classifyHTTP1HeaderOrderToken(key)
}
