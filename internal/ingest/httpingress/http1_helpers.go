package httpingress

func httpPathValid(b []byte) bool {
	for _, c := range b {
		if c == 0 || c < 0x20 || c > 0x7E {
			return false
		}
	}
	return len(b) > 0
}

var ingressPathValidFn func(method, path []byte) bool

func SetIngressPathValidFn(fn func(method, path []byte) bool) {
	ingressPathValidFn = fn
}

func http1IngressValid(method, path []byte) bool {
	if ingressPathValidFn == nil {
		return false
	}
	return ingressPathValidFn(method, path)
}

func PathHasPrefix(path []byte, prefix string) bool {
	return httpPathHasPrefix(path, prefix)
}

func BytesEqual(b []byte, s string) bool {
	return bytesEqual(b, s)
}

func HeadersComplete(data []byte) bool {
	return http1HeadersComplete(data)
}

func AssignHeader(req *Request, key, val []byte, hFlags *uint8, clValue *int) error {
	return http1AssignHeader(req, key, val, hFlags, clValue)
}

func AssignWireMetadataHeaders(req *Request, key, val []byte) {
	http1AssignWireMetadataHeaders(req, key, val)
}

func KeyMatchFold(key []byte, lit string) bool {
	return http1KeyMatchFold(key, lit)
}

func httpPathHasPrefix(path []byte, prefix string) bool {
	pl := len(prefix)
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

func parseTCPMSSHeader(b []byte) (uint16, bool) {
	n, ok := parseContentLengthStrict(b)
	if !ok || n < 0 || n > 65535 {
		return 0, false
	}
	return uint16(n), true
}

func parseTCPTTLHeader(b []byte) (uint8, bool) {
	n, ok := parseContentLengthStrict(b)
	if !ok || n < 0 || n > 255 {
		return 0, false
	}
	return uint8(n), true
}

func parseTCPWindowHeader(b []byte) (uint16, bool) {
	n, ok := parseContentLengthStrict(b)
	if !ok || n < 0 || n > 65535 {
		return 0, false
	}
	return uint16(n), true
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
	_ = key[off+3]
	return uint32(httpFold[key[off]]) |
		uint32(httpFold[key[off+1]])<<8 |
		uint32(httpFold[key[off+2]])<<16 |
		uint32(httpFold[key[off+3]])<<24
}

func foldKeyU64(key []byte, off int) uint64 {
	_ = key[off+7]
	return uint64(foldKeyU32(key, off)) |
		uint64(foldKeyU32(key, off+4))<<32
}
