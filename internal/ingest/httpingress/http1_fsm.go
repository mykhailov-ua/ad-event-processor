package httpingress

import (
	"unsafe"
)

// Header parse accumulates transfer/body flags in hFlags (not on Request) until body routing runs.
const (
	http1flChunkedTE uint8 = 1 << iota
	http1flHasTE
	http1flInvalidTE
	http1flCLSet
)

func ParseHTTP1(data []byte, maxBody int64, scratchPtr *[]byte) (int, Request, error) {
	var req Request
	n := len(data)
	if n == 0 {
		return 0, req, ErrIncomplete
	}
	_ = data[n-1]

	i, err := parseHTTP1RequestLine(data, n, &req)
	if err != nil {
		return 0, req, err
	}

	var hFlags uint8
	var clValue int
	i, hFlags, clValue, err = parseHTTP1Headers(data, i, n, &req, &hFlags, &clValue)
	if err != nil {
		return 0, req, err
	}

	// Track policy runs on headers only; chunked /track is rejected before any body bytes are consumed.
	if err := http1TrackEdgePolicy(&req, hFlags); err != nil {
		return 0, req, err
	}

	if hFlags&http1flChunkedTE != 0 {
		// RFC 7230: chunked and Content-Length together are invalid on any route.
		if hFlags&http1flCLSet != 0 {
			return 0, req, ErrInvalid
		}
		consumed, body, contentLen, err := ParseHTTP1ChunkedBody(data, i, maxBody, scratchPtr)
		if err != nil {
			return 0, req, err
		}
		req.Body = body
		req.ContentLength = contentLen
		req.HasContentLength = true
		return i + consumed, req, nil
	}

	if hFlags&http1flCLSet != 0 {
		if int64(clValue) > maxBody {
			return 0, req, ErrPayloadTooLarge
		}
		if i+clValue > n {
			return 0, req, ErrIncomplete
		}
		// Body aliases peek buffer until gnet PinParsedHTTPRequest copies on Tier B offload.
		if clValue > 0 {
			req.Body = data[i : i+clValue]
		}
		req.ContentLength = clValue
		req.HasContentLength = true
		return i + clValue, req, nil
	}

	// POST /track without Content-Length or chunked TE: ErrInvalid (implicit empty body not allowed).
	if isPOSTTrack(&req) {
		return 0, req, ErrInvalid
	}
	return i, req, nil
}

func parseHTTP1RequestLine(data []byte, n int, req *Request) (int, error) {
	if n < 22 {
		return 0, ErrIncomplete
	}
	// Fast path: constant compare for canonical POST /track line before generic request-line DFA.
	if n >= 22 && *(*[22]byte)(unsafe.Pointer(&data[0])) == trackReqLine {
		req.Method = data[0:4]
		req.Path = data[5:11]
		i := 22
		for i < n && data[i] != '\r' {
			i++
		}
		if i+1 >= n {
			return 0, ErrIncomplete
		}
		if data[i+1] != '\n' {
			return 0, ErrInvalid
		}
		return i + 2, nil
	}
	if n >= 29 && *(*[29]byte)(unsafe.Pointer(&data[0])) == openrtbBidReqLine {
		req.Method = data[0:4]
		req.Path = data[5:18]
		i := 29
		for i < n && data[i] != '\r' {
			i++
		}
		if i+1 >= n {
			return 0, ErrIncomplete
		}
		if data[i+1] != '\n' {
			return 0, ErrInvalid
		}
		return i + 2, nil
	}

	i := 0
	for i < n && data[i] != ' ' {
		if data[i] < 0x21 || data[i] > 0x7E {
			return 0, ErrInvalid
		}
		i++
	}
	if i == 0 || i > http1MaxMethodLen {
		return 0, ErrInvalid
	}
	req.Method = data[0:i]
	if i >= n || data[i] != ' ' {
		return 0, ErrInvalid
	}
	i++
	pathStart := i
	for i < n && data[i] != ' ' {
		if !httpPathValid([]byte{data[i]}) {
			return 0, ErrInvalid
		}
		i++
	}
	if i == pathStart || i-pathStart > http1MaxPathLen {
		return 0, ErrInvalid
	}
	req.Path = data[pathStart:i]
	if i+12 > n || data[i] != ' ' || data[i+1] != 'H' || data[i+2] != 'T' || data[i+3] != 'T' || data[i+4] != 'P' || data[i+5] != '/' {
		return 0, ErrInvalid
	}
	if !http1VersionValid(data[i+6 : i+12]) {
		return 0, ErrInvalid
	}
	if i+12 >= n || data[i+12] != '\r' {
		return 0, ErrInvalid
	}
	if i+13 >= n || data[i+13] != '\n' {
		return 0, ErrInvalid
	}
	if !http1IngressValid(req.Method, req.Path) {
		return 0, ErrInvalid
	}
	return i + 14, nil
}

func parseHTTP1Headers(data []byte, i, n int, req *Request, hFlags *uint8, clValue *int) (int, uint8, int, error) {
	var flags uint8
	var cl int
	recordHeaderOrder := http1PathRecordsHeaderOrder(req.Method, req.Path)
	for i < n {
		if i+1 < n && data[i] == '\r' && data[i+1] == '\n' {
			return i + 2, flags, cl, nil
		}
		lineStart := i
		colon := -1
		for i < n && data[i] != '\r' {
			if data[i] == ':' {
				if colon < 0 {
					colon = i
				}
			}
			i++
		}
		if colon < 0 {
			return 0, flags, cl, ErrInvalid
		}
		if i+1 >= n {
			return 0, flags, cl, ErrIncomplete
		}
		if data[i+1] != '\n' {
			return 0, flags, cl, ErrInvalid
		}

		key := trimHTTPKey(data[lineStart:colon])
		val := trimHTTPVal(data[colon+1 : i])
		if len(key) == 0 || len(key) > http1MaxHeaderNameLen || !httpTokenValid(key) {
			return 0, flags, cl, ErrInvalid
		}
		if len(val) > http1MaxHeaderValLen || !httpHeaderValValid(val) {
			return 0, flags, cl, ErrInvalid
		}
		if recordHeaderOrder {
			if token := classifyHTTP1HeaderOrderToken(key); token != http1HdrNone {
				recordHTTP1HeaderOrder(req, token)
			}
		}
		if err := http1AssignHeader(req, key, val, &flags, &cl); err != nil {
			return 0, flags, cl, err
		}
		i += 2
	}
	return 0, flags, cl, ErrIncomplete
}
