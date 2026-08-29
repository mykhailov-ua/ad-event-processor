package httpingress

type H2StaticEntry struct {
	name  string
	value string
}

var h2StaticTable = []H2StaticEntry{
	{":authority", ""},
	{":method", "GET"},
	{":method", "POST"},
	{":path", "/"},
	{":path", "/index.html"},
	{":scheme", "http"},
	{":scheme", "https"},
	{":status", "200"},
	{":status", "204"},
	{":status", "206"},
	{":status", "304"},
	{":status", "400"},
	{":status", "404"},
	{":status", "500"},
	{"accept-charset", ""},
	{"accept-encoding", "gzip, deflate"},
	{"accept-language", ""},
	{"accept-ranges", "bytes"},
	{"accept", ""},
	{"access-control-allow-origin", ""},
	{"age", ""},
	{"allow", ""},
	{"authorization", ""},
	{"cache-control", ""},
	{"content-disposition", ""},
	{"content-encoding", ""},
	{"content-language", ""},
	{"content-length", ""},
	{"content-location", ""},
	{"content-range", ""},
	{"content-type", "text/html; charset=utf-8"},
	{"content-type", "text/plain; charset=utf-8"},
	{"content-type", "application/json"},
	{"cookie", ""},
	{"date", ""},
	{"etag", ""},
	{"expect", ""},
	{"expires", ""},
	{"from", ""},
	{"host", ""},
	{"if-match", ""},
	{"if-modified-since", ""},
	{"if-none-match", ""},
	{"if-range", ""},
	{"if-unmodified-since", ""},
	{"last-modified", ""},
	{"link", ""},
	{"location", ""},
	{"max-forwards", ""},
	{"proxy-authenticate", ""},
	{"proxy-authorization", ""},
	{"range", ""},
	{"referer", ""},
	{"refresh", ""},
	{"retry-after", ""},
	{"server", ""},
	{"set-cookie", ""},
	{"strict-transport-security", ""},
	{"transfer-encoding", ""},
	{"user-agent", ""},
	{"vary", ""},
	{"via", ""},
	{"www-authenticate", ""},
}

var (
	h2StaticNameB  [][]byte
	h2StaticValueB [][]byte
)

func init() {
	n := len(h2StaticTable)
	h2StaticNameB = make([][]byte, n)
	h2StaticValueB = make([][]byte, n)
	for i, e := range h2StaticTable {
		h2StaticNameB[i] = []byte(e.name)
		h2StaticValueB[i] = []byte(e.value)
	}
}

func h2StaticNameValue(index int) (name, value []byte, ok bool) {
	if index < 1 || index > len(h2StaticTable) {
		return nil, nil, false
	}
	i := index - 1
	return h2StaticNameB[i], h2StaticValueB[i], true
}

func h2DecodeString(data []byte, off int) (val []byte, next int, err error) {
	n := len(data)
	if off >= n {
		return nil, 0, ErrIncomplete
	}
	_ = data[n-1]
	huff := data[off]&0x80 != 0
	strLen, off, err := h2DecodeInt(data, off, 7, 0x7f)
	if err != nil {
		return nil, 0, err
	}
	if strLen < 0 || off+strLen > n {
		return nil, 0, ErrIncomplete
	}
	raw := data[off : off+strLen]
	off += strLen
	if huff {
		return nil, 0, ErrInvalid
	}
	return raw, off, nil
}

func h2DecodeHeadersBlock(block []byte, req *Request) error {
	var hFlags uint8
	var clValue int
	off := 0
	n := len(block)
	for off < n {
		_ = block[n-1]
		b := block[off]
		if b&0x80 != 0 {
			idx, next, err := h2DecodeInt(block, off, 7, 0x7f)
			if err != nil {
				return err
			}
			off = next
			name, val, ok := h2StaticNameValue(idx)
			if !ok {
				return ErrInvalid
			}
			if err := h2AssignHeader(req, name, val, &hFlags, &clValue); err != nil {
				return err
			}
			continue
		}
		if b&0x40 != 0 {
			return ErrInvalid
		}
		if b&0x20 != 0 {
			return ErrInvalid
		}
		if b&0x10 != 0 {
			nameIdx, next, err := h2DecodeInt(block, off, 4, 0x0f)
			if err != nil {
				return err
			}
			off = next
			var name []byte
			if nameIdx > 0 {
				var ok bool
				name, _, ok = h2StaticNameValue(nameIdx)
				if !ok {
					return ErrInvalid
				}
			} else {
				name, off, err = h2DecodeString(block, off)
				if err != nil {
					return err
				}
			}
			var val []byte
			val, off, err = h2DecodeString(block, off)
			if err != nil {
				return err
			}
			if err := h2AssignHeader(req, name, val, &hFlags, &clValue); err != nil {
				return err
			}
			continue
		}
		if b&0x80 == 0 && b&0x40 == 0 && b&0x20 == 0 && b&0x10 == 0 {
			nameIdx, next, err := h2DecodeInt(block, off, 4, 0x0f)
			if err != nil {
				return err
			}
			off = next
			var name []byte
			if nameIdx > 0 {
				var ok bool
				name, _, ok = h2StaticNameValue(nameIdx)
				if !ok {
					return ErrInvalid
				}
			} else {
				name, off, err = h2DecodeString(block, off)
				if err != nil {
					return err
				}
			}
			var val []byte
			val, off, err = h2DecodeString(block, off)
			if err != nil {
				return err
			}
			if err := h2AssignHeader(req, name, val, &hFlags, &clValue); err != nil {
				return err
			}
			continue
		}
		return ErrInvalid
	}
	if hFlags&http1flHasTE != 0 {
		return ErrInvalid
	}
	return nil
}

func h2AssignHeader(req *Request, key, val []byte, hFlags *uint8, clValue *int) error {
	if len(key) > 0 && key[0] == ':' {
		switch len(key) {
		case 7:
			if bytesEqual(key, ":method") {
				req.Method = val
				recordH2PseudoHeader(req, h2PseudoMethod)
				return nil
			}
		case 5:
			if bytesEqual(key, ":path") {
				req.Path = val
				recordH2PseudoHeader(req, h2PseudoPath)
				if len(req.Method) > 0 && !http1IngressValid(req.Method, req.Path) {
					return ErrInvalid
				}
				return nil
			}
		}
		if bytesEqual(key, ":authority") {
			req.Host = val
			recordH2PseudoHeader(req, h2PseudoAuthority)
			return nil
		}
		if bytesEqual(key, ":scheme") {
			recordH2PseudoHeader(req, h2PseudoScheme)
			return nil
		}
		return ErrInvalid
	}
	if h2KeyHasUppercase(key) || h2ForbiddenH1HeaderName(key) {
		markH2DowngradeArtifact(req)
	}
	var folded [http1MaxHeaderNameLen]byte
	if len(key) > len(folded) {
		return ErrInvalid
	}
	for i, c := range key {
		folded[i] = httpFold[c]
	}
	return http1AssignHeader(req, folded[:len(key)], val, hFlags, clValue)
}

func h2EncodeString(dst []byte, off int, val []byte) int {
	off = h2EncodeInt(dst, off, len(val), 7, 0x7f)
	if off+len(val) > len(dst) {
		return off
	}
	copy(dst[off:], val)
	return off + len(val)
}

var (
	h2ServerSettings = []byte{
		0x00, 0x00, 0x0c, h2FrameSettings, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x03, 0x00, 0x00, 0x00, 0x80,
		0x00, 0x02, 0x00, 0x00, 0x00, 0x00,
	}
	h2SettingsACK = []byte{
		0x00, 0x00, 0x00, h2FrameSettings, 0x01, 0x00, 0x00, 0x00, 0x00,
	}
	h2ConnBootstrap = append(append([]byte(nil), h2ServerSettings...), h2SettingsACK...)
)

func h2EncodeStatusResponse(dst []byte, streamID uint32, status int, contentType, body []byte) int {
	blockOff := h2FrameHeaderSize
	block := dst[blockOff : blockOff+256]
	boff := 0
	block[boff] = 0x00
	boff++
	boff = h2EncodeInt(block, boff, 8, 4, 0x0f)
	boff = h2EncodeString(block, boff, []byte(h2StatusString(status)))
	if len(contentType) > 0 {
		block[boff] = 0x00
		boff++
		boff = h2EncodeInt(block, boff, 31, 4, 0x0f)
		if len(contentType) != 16 || !bytesEqual(contentType, "application/json") {
			boff = h2EncodeString(block, boff, contentType)
		}
	}
	blockLen := boff
	hdrFlags := h2FlagEndHeaders
	if len(body) == 0 {
		hdrFlags |= h2FlagEndStream
	}
	encodeH2FrameHeader(dst, uint32(blockLen), h2FrameHeaders, hdrFlags, streamID)
	copy(dst[h2FrameHeaderSize:], block[:blockLen])
	n := h2FrameHeaderSize + blockLen
	if len(body) > 0 {
		n = h2EncodeDataFrame(dst, n, streamID, body, true)
	}
	return n
}

func h2StatusString(code int) string {
	switch code {
	case 200:
		return "200"
	case 202:
		return "202"
	case 204:
		return "204"
	case 400:
		return "400"
	case 404:
		return "404"
	case 413:
		return "413"
	case 429:
		return "429"
	case 500:
		return "500"
	case 503:
		return "503"
	default:
		return "500"
	}
}

func h2EncodeDataFrame(dst []byte, off int, streamID uint32, payload []byte, endStream bool) int {
	flags := byte(0)
	if endStream {
		flags = h2FlagEndStream
	}
	encodeH2FrameHeader(dst[off:], uint32(len(payload)), h2FrameData, flags, streamID)
	off += h2FrameHeaderSize
	off += copy(dst[off:], payload)
	return off
}

func h2WrapH1Response(dst []byte, streamID uint32, h1 []byte) (int, error) {
	status, body, contentType, ok := parseH1ResponseForH2(h1)
	if !ok {
		return 0, ErrInvalid
	}
	return h2EncodeStatusResponse(dst, streamID, status, contentType, body), nil
}

func parseH1ResponseForH2(h1 []byte) (status int, body, contentType []byte, ok bool) {
	if len(h1) < 12 || !bytesEqual(h1[:5], "HTTP/") {
		return 0, nil, nil, false
	}
	code := 0
	digits := 0
	for i := 5; i < len(h1) && i < 24; i++ {
		if h1[i] == ' ' {
			if digits > 0 {
				break
			}
			continue
		}
		if h1[i] >= '0' && h1[i] <= '9' {
			code = code*10 + int(h1[i]-'0')
			digits++
			continue
		}
		if digits > 0 {
			break
		}
	}
	if digits == 0 {
		return 0, nil, nil, false
	}
	hdrEnd := -1
	for i := 0; i+3 < len(h1); i++ {
		if h1[i] == '\r' && h1[i+1] == '\n' && h1[i+2] == '\r' && h1[i+3] == '\n' {
			hdrEnd = i + 4
			break
		}
	}
	if hdrEnd < 0 {
		return 0, nil, nil, false
	}
	body = h1[hdrEnd:]
	line := 0
	for lineStart := 0; lineStart < hdrEnd-1; {
		lineEnd := lineStart
		for lineEnd+1 < hdrEnd && (h1[lineEnd] != '\r' || h1[lineEnd+1] != '\n') {
			lineEnd++
		}
		if line > 0 {
			colon := -1
			for j := lineStart; j < lineEnd; j++ {
				if h1[j] == ':' {
					colon = j
					break
				}
			}
			if colon > 0 {
				key := trimHTTPKey(h1[lineStart:colon])
				if len(key) == 12 && foldKeyU64(key, 0) == 0x2d746e65746e6f63 && foldKeyU32(key, 8) == 0x65707974 {
					contentType = trimHTTPVal(h1[colon+1 : lineEnd])
				}
			}
		}
		line++
		lineStart = lineEnd + 2
	}
	return code, body, contentType, true
}

const h2MaxIntContinuationOctets = 8

func h2DecodeInt(data []byte, off int, prefixBits byte, prefixMask byte) (value int, next int, err error) {
	n := len(data)
	if off >= n {
		return 0, 0, ErrIncomplete
	}
	_ = data[n-1]
	b := data[off]
	val := int(b & prefixMask)
	off++
	if val < int(prefixMask) {
		return val, off, nil
	}
	mult := 1
	continuations := 0
	for off < n {
		b = data[off]
		off++
		val += int(b&0x7f) * mult
		mult <<= 7
		if b < 0x80 {
			return val, off, nil
		}
		continuations++
		if continuations > h2MaxIntContinuationOctets {
			return 0, 0, ErrInvalid
		}
		if mult > 1<<30 {
			return 0, 0, ErrInvalid
		}
	}
	return 0, 0, ErrIncomplete
}

func h2EncodeInt(dst []byte, off int, value int, prefixBits byte, prefixMask byte) int {
	if value < int(prefixMask) {
		if off >= len(dst) {
			return off
		}
		dst[off] = (dst[off] &^ prefixMask) | byte(value)
		return off + 1
	}
	if off >= len(dst) {
		return off
	}
	dst[off] |= prefixMask
	off++
	value -= int(prefixMask)
	for value >= 0x80 {
		if off >= len(dst) {
			return off
		}
		dst[off] = byte(value%0x80 + 0x80)
		off++
		value /= 0x80
	}
	if off < len(dst) {
		dst[off] = byte(value)
		off++
	}
	return off
}
