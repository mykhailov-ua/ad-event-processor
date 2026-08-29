package httpingress

func teValueOnlyChunked(val []byte) bool {
	for i := range val {
		c := val[i]
		if c < 0x20 && c != ' ' && c != '\t' {
			return false
		}
	}
	if !teValueHasChunked(val) {
		return false
	}
	i := 0
	found := false
	for i < len(val) {
		for i < len(val) && (val[i] == ' ' || val[i] == '\t' || val[i] == ',') {
			i++
		}
		if i >= len(val) {
			break
		}
		start := i
		for i < len(val) && val[i] != ',' {
			i++
		}
		token := trimHTTPVal(val[start:i])
		if len(token) == 0 {
			continue
		}
		if len(token) == 7 &&
			foldKeyU32(token, 0) == 0x6e756863 &&
			httpFold[token[4]] == 'k' &&
			httpFold[token[5]] == 'e' &&
			httpFold[token[6]] == 'd' {
			if found {
				return false
			}
			found = true
			continue
		}
		return false
	}
	return found
}

const (
	chunkScratchInitCap   = 4096
	chunkScratchRetainCap = 64 << 10
)

func growChunkScratch(scratchPtr *[]byte, totalLen int) []byte {
	if scratchPtr == nil {
		return make([]byte, totalLen)
	}
	buf := *scratchPtr
	if cap(buf) < totalLen {
		buf = make([]byte, totalLen)
		*scratchPtr = buf
	} else {
		buf = buf[:totalLen]
	}
	return buf
}

func ResetChunkScratch(scratchPtr *[]byte) {
	if scratchPtr == nil {
		return
	}
	if cap(*scratchPtr) > chunkScratchRetainCap {
		*scratchPtr = make([]byte, 0, chunkScratchInitCap)
		return
	}
	*scratchPtr = (*scratchPtr)[:0]
}

func ParseHTTP1ChunkedBody(data []byte, off int, maxBody int64, scratchPtr *[]byte) (consumed int, body []byte, contentLen int, err error) {
	n := len(data)
	pos := off
	totalLen := 0
	firstStart := -1
	contiguousEnd := -1
	fragmented := false
	scratchLen := 0

	for {
		if pos >= n {
			return 0, nil, 0, ErrIncomplete
		}
		size, lineEnd, perr := ParseChunkSizeLine(data, pos, n)
		if perr != nil {
			return 0, nil, 0, perr
		}
		pos = lineEnd

		if size == 0 {
			pos, perr = skipHTTP1ChunkTrailers(data, pos, n)
			if perr != nil {
				return 0, nil, 0, perr
			}
			if totalLen == 0 {
				return pos, nil, 0, nil
			}
			if !fragmented && firstStart >= 0 && contiguousEnd == firstStart+totalLen {
				return pos, data[firstStart:contiguousEnd], totalLen, nil
			}
			scratch := growChunkScratch(scratchPtr, scratchLen)
			return pos, scratch[:scratchLen], totalLen, nil
		}

		if int64(totalLen+size) > maxBody {
			return 0, nil, 0, ErrPayloadTooLarge
		}
		if pos+size+2 > n {
			return 0, nil, 0, ErrIncomplete
		}
		if data[pos+size] != '\r' || data[pos+size+1] != '\n' {
			return 0, nil, 0, ErrInvalid
		}

		if fragmented {
			scratch := growChunkScratch(scratchPtr, scratchLen+size)
			copy(scratch[scratchLen:], data[pos:pos+size])
			scratchLen += size
		} else if firstStart >= 0 && pos != contiguousEnd {
			fragmented = true
			prefixLen := contiguousEnd - firstStart
			scratch := growChunkScratch(scratchPtr, prefixLen+size)
			copy(scratch, data[firstStart:contiguousEnd])
			copy(scratch[prefixLen:], data[pos:pos+size])
			scratchLen = prefixLen + size
		} else if firstStart < 0 {
			firstStart = pos
			contiguousEnd = pos + size
		} else {
			contiguousEnd = pos + size
		}
		totalLen += size
		pos += size + 2
	}
}

func ParseChunkSizeLine(data []byte, pos, n int) (size int, next int, err error) {
	if pos >= n {
		return 0, 0, ErrIncomplete
	}
	hasDigit := false
	i := pos
	for i < n {
		b := data[i]
		if b == '\r' {
			if i+1 >= n {
				return 0, 0, ErrIncomplete
			}
			if data[i+1] != '\n' {
				return 0, 0, ErrInvalid
			}
			if !hasDigit {
				return 0, 0, ErrInvalid
			}
			return size, i + 2, nil
		}
		if b == ';' {
			return 0, 0, ErrInvalid
		}
		if b >= '0' && b <= '9' {
			hasDigit = true
			d := int(b - '0')
			if size > (1<<60)/16 || size*16 > (1<<60)-d {
				return 0, 0, ErrInvalid
			}
			size = size*16 + d
			i++
			continue
		}
		if b >= 'a' && b <= 'f' {
			hasDigit = true
			d := int(b - 'a' + 10)
			if size > (1<<60)/16 || size*16 > (1<<60)-d {
				return 0, 0, ErrInvalid
			}
			size = size*16 + d
			i++
			continue
		}
		if b >= 'A' && b <= 'F' {
			hasDigit = true
			d := int(b - 'A' + 10)
			if size > (1<<60)/16 || size*16 > (1<<60)-d {
				return 0, 0, ErrInvalid
			}
			size = size*16 + d
			i++
			continue
		}
		if b == ' ' || b == '\t' {
			i++
			continue
		}
		return 0, 0, ErrInvalid
	}
	return 0, 0, ErrIncomplete
}

func skipHTTP1ChunkTrailers(data []byte, pos, n int) (int, error) {
	for {
		if pos >= n {
			return 0, ErrIncomplete
		}
		if data[pos] == '\r' {
			if pos+1 >= n {
				return 0, ErrIncomplete
			}
			if data[pos+1] != '\n' {
				return 0, ErrInvalid
			}
			return pos + 2, nil
		}
		for pos < n && data[pos] != '\r' {
			if data[pos] == 0 {
				return 0, ErrInvalid
			}
			pos++
		}
		if pos+1 >= n {
			return 0, ErrIncomplete
		}
		if data[pos+1] != '\n' {
			return 0, ErrInvalid
		}
		pos += 2
	}
}
