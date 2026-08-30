package httpingress

// Chaos/load harness only: fixed 31-byte name, case-folded compare. Not an auth bypass in prod edge.
func http1MatchForceSafeHeader(key []byte) bool {
	if len(key) != 31 {
		return false
	}
	const lit = "x-ad-event-processor-force-safe"
	for i := range 31 {
		if httpFold[key[i]] != lit[i] {
			return false
		}
	}
	return true
}

func http1ForceSafeValue(val []byte) bool {
	if len(val) == 1 && val[0] == '1' {
		return true
	}
	if len(val) == 4 && httpFold[val[0]] == 't' && httpFold[val[1]] == 'r' &&
		httpFold[val[2]] == 'u' && httpFold[val[3]] == 'e' {
		return true
	}
	return false
}

// Second Transfer-Encoding header is ErrInvalid. Only a lone "chunked" token (no gzip, deflate, etc.)
// sets http1flChunkedTE; anything else marks http1flInvalidTE for later rejection.
func http1AssignTransferEncoding(hFlags *uint8, val []byte) error {
	if *hFlags&http1flHasTE != 0 {
		return ErrInvalid
	}
	*hFlags |= http1flHasTE
	if teValueOnlyChunked(val) {
		*hFlags |= http1flChunkedTE
	} else {
		*hFlags |= http1flInvalidTE
	}
	return nil
}

// Canonical POST /track wire for TestChaos_CrossHop_NginxGnet; disposition must match edge Lua + gnet.
var NginxTrackCorpus = []byte(
	"POST /track HTTP/1.1\r\n" +
		"Host: edge.local\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 69\r\n" +
		"X-Forwarded-For: 203.0.113.10\r\n" +
		"X-Real-IP: 203.0.113.10\r\n" +
		"User-Agent: Mozilla/5.0\r\n" +
		"Accept: application/json\r\n" +
		"Accept-Language: en-US\r\n" +
		"X-TLS-Hash: abc123def456\r\n" +
		"Sec-CH-UA: \"Chromium\";v=\"120\"\r\n" +
		"Connection: keep-alive\r\n" +
		"\r\n" +
		`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}`,
)

const http1MaxBufferedOverhead = MaxBufferedOverhead

func http1HeadersComplete(data []byte) bool {
	for i := 0; i+3 < len(data); i++ {
		if data[i] == '\r' && data[i+1] == '\n' && data[i+2] == '\r' && data[i+3] == '\n' {
			return true
		}
	}
	return false
}

// Edge/nginx parity for POST /track: require Content-Length, forbid chunked TE before body read.
// Rejects TE.TE obfuscation and slow-body patterns; OpenRTB bid keeps chunked (http1_fsm.go).
func http1TrackEdgePolicy(req *Request, hFlags uint8) error {
	if req == nil || !isPOSTTrack(req) {
		return nil
	}
	if hFlags&http1flChunkedTE != 0 {
		return ErrInvalid
	}
	if hFlags&http1flCLSet == 0 {
		return ErrInvalid
	}
	return nil
}

func isPOSTTrack(req *Request) bool {
	return len(req.Method) == 4 &&
		req.Method[0] == 'P' && req.Method[1] == 'O' && req.Method[2] == 'S' && req.Method[3] == 'T' &&
		httpPathHasPrefix(req.Path, "/track")
}
