package ingestion

import "unsafe"

const (
	http1MaxMethodLen     = 16
	http1MaxPathLen       = 2048
	http1MaxHeaderNameLen = 256
	http1MaxHeaderValLen  = 1024
)

var (
	httpFold [256]byte

	trackReqLine = [22]byte{
		'P', 'O', 'S', 'T', ' ', '/', 't', 'r', 'a', 'c', 'k', ' ',
		'H', 'T', 'T', 'P', '/', '1', '.', '1', '\r', '\n',
	}
	openrtbBidReqLine = [29]byte{
		'P', 'O', 'S', 'T', ' ', '/', 'o', 'p', 'e', 'n', 'r', 't', 'b', '/', 'b', 'i', 'd', ' ',
		'H', 'T', 'T', 'P', '/', '1', '.', '1', '\r', '\n',
	}
)

func init() {
	initHTTP1ValidateTables()
	for i := range 256 {
		httpFold[i] = byte(i)
	}
	for i := 'A'; i <= 'Z'; i++ {
		httpFold[i] = byte(i + ('a' - 'A'))
	}
}

const (
	http1flChunkedTE uint8 = 1 << iota
	http1flHasTE
	http1flInvalidTE
	http1flCLSet
)

func parseHTTP1(data []byte, maxBody int64, scratchPtr *[]byte) (int, parsedHTTPRequest, error) {
	var req parsedHTTPRequest
	n := len(data)
	if n == 0 {
		return 0, req, errIncompleteRequest
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

	if hFlags&http1flInvalidTE != 0 {
		return 0, req, errInvalidRequest
	}

	if err := http1TrackEdgePolicy(&req, hFlags); err != nil {
		return 0, req, err
	}

	if hFlags&http1flChunkedTE != 0 {
		if hFlags&http1flCLSet != 0 {
			return 0, req, errInvalidRequest
		}
		consumed, body, cl, err := parseHTTP1ChunkedBody(data, i, maxBody, scratchPtr)
		if err != nil {
			return 0, req, err
		}
		req.Body = body
		req.ContentLength = cl
		req.HasContentLength = true
		return consumed, req, nil
	}

	if hFlags&http1flHasTE != 0 {
		return 0, req, errInvalidRequest
	}

	if req.HasContentLength && int64(req.ContentLength) > maxBody {
		return 0, req, errPayloadTooLarge
	}
	total := i + req.ContentLength
	if n < total {
		return 0, req, errIncompleteRequest
	}
	if req.ContentLength > 0 {
		req.Body = data[i : i+req.ContentLength]
	}
	return total, req, nil
}

func parseHTTP1RequestLine(data []byte, n int, req *parsedHTTPRequest) (int, error) {
	if req == nil {
		return 0, errInvalidRequest
	}
	i := 0
	if n >= 22 && *(*[22]byte)(unsafe.Pointer(&data[0])) == trackReqLine {
		req.Method = data[:4]
		req.Path = data[5:11]
		return 22, nil
	}
	if n >= 29 && *(*[29]byte)(unsafe.Pointer(&data[0])) == openrtbBidReqLine {
		req.Method = data[:4]
		req.Path = data[5:18]
		return 29, nil
	}

	sp1, sp2 := -1, -1
	for i < n {
		b := data[i]
		if b == '\r' {
			if i+1 >= n {
				return 0, errIncompleteRequest
			}
			if data[i+1] != '\n' {
				return 0, errInvalidRequest
			}
			if sp1 < 0 || sp2 < 0 {
				return 0, errInvalidRequest
			}
			req.Method = data[:sp1]
			req.Path = data[sp1+1 : sp2]
			if len(req.Method) > http1MaxMethodLen || len(req.Path) > http1MaxPathLen {
				return 0, errInvalidRequest
			}
			if !httpTokenValid(req.Method) || !httpPathValid(req.Path) || !http1VersionValid(data[sp2+1:i]) {
				return 0, errInvalidRequest
			}
			if !http1IngressValid(req.Method, req.Path) {
				return 0, errInvalidRequest
			}
			return i + 2, nil
		}
		if b == 0 || (b < 0x20 && b != '\t') {
			return 0, errInvalidRequest
		}
		if b == ' ' {
			if sp1 < 0 {
				sp1 = i
			} else if sp2 < 0 {
				sp2 = i
			}
			i++
			continue
		}
		i++
	}
	return 0, errIncompleteRequest
}

func parseHTTP1Headers(data []byte, i, n int, req *parsedHTTPRequest, hFlags *uint8, clValue *int) (int, uint8, int, error) {
	if req == nil || hFlags == nil || clValue == nil {
		return 0, 0, 0, errInvalidRequest
	}
	flags := *hFlags
	cl := *clValue

	for {
		if i >= n {
			return 0, flags, cl, errIncompleteRequest
		}
		if data[i] == '\r' {
			if i+1 >= n {
				return 0, flags, cl, errIncompleteRequest
			}
			if data[i+1] != '\n' {
				return 0, flags, cl, errInvalidRequest
			}
			return i + 2, flags, cl, nil
		}

		lineStart := i
		colon := -1
		for i < n {
			b := data[i]
			if b == 0 || (b < 0x20 && b != '\t') {
				return 0, flags, cl, errInvalidRequest
			}
			if b == ':' {
				colon = i
				i++
				break
			}
			if b == '\r' {
				return 0, flags, cl, errInvalidRequest
			}
			i++
		}
		if colon < 0 {
			return 0, flags, cl, errIncompleteRequest
		}

		for i < n && data[i] != '\r' {
			if data[i] == 0 {
				return 0, flags, cl, errInvalidRequest
			}
			i++
		}
		if i+1 >= n {
			return 0, flags, cl, errIncompleteRequest
		}
		if data[i+1] != '\n' {
			return 0, flags, cl, errInvalidRequest
		}

		key := trimHTTPKey(data[lineStart:colon])
		val := trimHTTPVal(data[colon+1 : i])
		if len(key) == 0 || len(key) > http1MaxHeaderNameLen || !httpTokenValid(key) {
			return 0, flags, cl, errInvalidRequest
		}
		if len(val) > http1MaxHeaderValLen || !httpHeaderValValid(val) {
			return 0, flags, cl, errInvalidRequest
		}
		if err := http1AssignHeader(req, key, val, &flags, &cl); err != nil {
			return 0, flags, cl, err
		}
		i += 2
	}
}

func httpPathValid(b []byte) bool {
	for _, c := range b {
		if c == 0 || c < 0x20 || c > 0x7E {
			return false
		}
	}
	return len(b) > 0
}

func http1IngressValid(method, path []byte) bool {
	if len(method) == 4 && method[0] == 'P' && method[1] == 'O' && method[2] == 'S' && method[3] == 'T' {
		return httpPathHasPrefix(path, "/track") || httpPathHasPrefix(path, "/openrtb/bid") || httpPathHasPrefix(path, "/tg/bid")
	}
	if len(method) == 7 && method[0] == 'O' && method[1] == 'P' && method[2] == 'T' &&
		method[3] == 'I' && method[4] == 'O' && method[5] == 'N' && method[6] == 'S' {
		return bytesEqual(path, "/track")
	}
	if len(method) == 3 && method[0] == 'G' && method[1] == 'E' && method[2] == 'T' {
		return bytesEqual(path, "/health") ||
			bytesEqual(path, "/healthz") ||
			bytesEqual(path, "/ready") ||
			bytesEqual(path, "/readyz") ||
			bytesEqual(path, "/metrics") ||
			httpPathHasPrefix(path, safePageStubPathPrefix) ||
			httpPathHasPrefix(path, "/click") ||
			httpPathHasPrefix(path, tgPathClick) ||
			httpPathHasPrefix(path, tgPathImpression)
	}
	return false
}

func httpPathHasPrefix(path []byte, prefix string) bool {
	p := []byte(prefix)
	pl := len(p)
	pn := len(path)
	if pn < pl {
		return false
	}
	_ = path[pn-1]
	if !bytesEqual(path[:pl], prefix) {
		return false
	}
	if pn == pl {
		return true
	}
	switch path[pl] {
	case '?', '/':
		return true
	default:
		return false
	}
}

func bytesEqual(b []byte, s string) bool {
	if len(b) != len(s) {
		return false
	}
	for i := range b {
		if b[i] != s[i] {
			return false
		}
	}
	return true
}

func teValueHasChunked(val []byte) bool {
	vn := len(val)
	if vn == 0 {
		return false
	}
	_ = val[vn-1]
	i := 0
	for i < vn {
		for i < vn && (val[i] == ' ' || val[i] == '\t' || val[i] == ',') {
			i++
		}
		if i >= vn {
			break
		}
		start := i
		for i < vn && val[i] != ',' {
			i++
		}
		token := val[start:i]
		if len(token) == 7 &&
			foldKeyU32(token, 0) == 0x6e756863 &&
			httpFold[token[4]] == 'k' &&
			httpFold[token[5]] == 'e' &&
			httpFold[token[6]] == 'd' {
			return true
		}
	}
	return false
}

func http1VersionValid(b []byte) bool {
	return len(b) == 8 &&
		foldKeyU32(b, 0) == 0x70747468 &&
		foldKeyU32(b, 4) == 0x312e312f
}

func parseContentLengthStrict(b []byte) (int, bool) {
	if len(b) == 0 {
		return 0, false
	}
	val := 0
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, false
		}
		if val > (1<<31-1)/10 {
			return 0, false
		}
		next := val*10 + int(c-'0')
		if next < val {
			return 0, false
		}
		val = next
	}
	return val, true
}

func trimHTTPKey(b []byte) []byte {
	end := len(b)
	for end > 0 && (b[end-1] == ' ' || b[end-1] == '\t') {
		end--
	}
	return b[:end]
}

func trimHTTPVal(b []byte) []byte {
	start := 0
	for start < len(b) && (b[start] == ' ' || b[start] == '\t') {
		start++
	}
	end := len(b)
	for end > start && (b[end-1] == ' ' || b[end-1] == '\t') {
		end--
	}
	return b[start:end]
}

func foldKeyU32(key []byte, off int) uint32 {
	_ = key[off+3] // BCE: callers pass off with len(key) >= off+4
	return uint32(httpFold[key[off]]) |
		uint32(httpFold[key[off+1]])<<8 |
		uint32(httpFold[key[off+2]])<<16 |
		uint32(httpFold[key[off+3]])<<24
}

func foldKeyU64(key []byte, off int) uint64 {
	_ = key[off+7] // BCE: callers pass off with len(key) >= off+8
	return uint64(foldKeyU32(key, off)) |
		uint64(foldKeyU32(key, off+4))<<32
}

func http1AssignHeader(req *parsedHTTPRequest, key, val []byte, hFlags *uint8, clValue *int) error {
	kl := len(key)
	if kl == 2 && httpFold[key[0]] == 't' && httpFold[key[1]] == 'e' {
		return http1AssignTransferEncoding(hFlags, val)
	}
	if kl == 4 {
		if httpFold[key[0]] == 'h' && httpFold[key[1]] == 'o' && httpFold[key[2]] == 's' && httpFold[key[3]] == 't' {
			req.Host = val
		}
		return nil
	}
	if kl < 6 {
		return nil
	}
	_ = key[kl-1]

	switch kl {
	case 6:
		if foldKeyU32(key, 0) == 0x65636361 && httpFold[key[4]] == 'p' && httpFold[key[5]] == 't' {
			req.Accept = val
		} else if foldKeyU32(key, 0) == 0x6769726f && httpFold[key[4]] == 'i' && httpFold[key[5]] == 'n' {
			req.Origin = val
		}
	case 9:
		switch foldKeyU32(key, 0) {
		case 0x65722d78:
			if httpFold[key[4]] == 'a' && httpFold[key[5]] == 'l' && httpFold[key[6]] == '-' &&
				httpFold[key[7]] == 'i' && httpFold[key[8]] == 'p' {
				if len(req.ClientIP) == 0 {
					req.ClientIP = val
				}
			}
		case 0x2d636573:
			if foldKeyU32(key, 4) == 0x752d6863 && httpFold[key[8]] == 'a' {
				req.SecCHUA = val
			}
		}
	case 10:
		switch foldKeyU32(key, 0) {
		case 0x72657375:
			if foldKeyU32(key, 4) == 0x6567612d && httpFold[key[8]] == 'n' && httpFold[key[9]] == 't' {
				req.UserAgent = val
			}
		case 0x6c742d78:
			if foldKeyU32(key, 4) == 0x61682d73 && httpFold[key[8]] == 's' && httpFold[key[9]] == 'h' {
				req.TLSHash = val
			}
		}
	case 12:
		if foldKeyU64(key, 0) == 0x2d746e65746e6f63 && foldKeyU32(key, 8) == 0x65707974 {
			req.ContentType = val
		}
	case 14:
		if foldKeyU64(key, 0) == 0x2d746e65746e6f63 && foldKeyU32(key, 8) == 0x676e656c &&
			httpFold[key[12]] == 't' && httpFold[key[13]] == 'h' {
			cl, ok := parseContentLengthStrict(val)
			if !ok {
				return errInvalidRequest
			}
			if *hFlags&http1flCLSet != 0 && *clValue != cl {
				return errInvalidRequest
			}
			*hFlags |= http1flCLSet
			*clValue = cl
			req.ContentLength = cl
			req.HasContentLength = true
		}
	case 15:
		switch foldKeyU32(key, 0) {
		case 0x6f662d78:
			if foldKeyU64(key, 4) == 0x2d64656472617772 && httpFold[key[12]] == 'f' &&
				httpFold[key[13]] == 'o' && httpFold[key[14]] == 'r' {
				req.ClientIP = val
			}
		case 0x65636361:
			if key[6] == '-' && httpFold[key[7]] == 'l' {
				req.AcceptLang = val
			} else if key[6] == '-' && httpFold[key[7]] == 'e' {
				req.AcceptEncoding = val
			}
		}
	case 17:
		if foldKeyU64(key, 0) == 0x726566736e617274 && foldKeyU64(key, 8) == 0x6e69646f636e652d &&
			httpFold[key[16]] == 'g' {
			return http1AssignTransferEncoding(hFlags, val)
		}
	case 21:
		if http1MatchForceSafeHeader(key) && http1ForceSafeValue(val) {
			req.ForceSafe = true
		}
	}
	return nil
}

func http1MatchForceSafeHeader(key []byte) bool {
	if len(key) != 21 {
		return false
	}
	const lit = "x-bidshard-force-safe"
	for i := 0; i < 21; i++ {
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

func http1AssignTransferEncoding(hFlags *uint8, val []byte) error {
	if *hFlags&http1flHasTE != 0 {
		return errInvalidRequest
	}
	*hFlags |= http1flHasTE
	if teValueOnlyChunked(val) {
		*hFlags |= http1flChunkedTE
	} else {
		*hFlags |= http1flInvalidTE
	}
	return nil
}
