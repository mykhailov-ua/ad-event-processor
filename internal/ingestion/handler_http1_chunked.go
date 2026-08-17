package ingestion

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
	chunkScratchRetainCap = 64 << 10 // align with maxPoolObjectSize
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

// resetChunkScratch drops oversized per-connection chunked reassembly buffers (PS-H03).
func resetChunkScratch(scratchPtr *[]byte) {
	if scratchPtr == nil {
		return
	}
	if cap(*scratchPtr) > chunkScratchRetainCap {
		*scratchPtr = make([]byte, 0, chunkScratchInitCap)
		return
	}
	*scratchPtr = (*scratchPtr)[:0]
}

func parseHTTP1ChunkedBody(data []byte, off int, maxBody int64, scratchPtr *[]byte) (consumed int, body []byte, contentLen int, err error) {
	n := len(data)
	pos := off
	totalLen := 0
	firstStart := -1
	contiguousEnd := -1

	for {
		if pos >= n {
			return 0, nil, 0, errIncompleteRequest
		}
		size, lineEnd, perr := parseChunkSizeLine(data, pos, n)
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
			if firstStart >= 0 && contiguousEnd == firstStart+totalLen {
				return pos, data[firstStart:contiguousEnd], totalLen, nil
			}
			copyScratch := growChunkScratch(scratchPtr, totalLen)
			rpos := off
			acc := 0
			for {
				chunkSize, next, perr := parseChunkSizeLine(data, rpos, n)
				if perr != nil {
					return 0, nil, 0, perr
				}
				if chunkSize == 0 {
					break
				}
				chunkData := data[next : next+chunkSize]
				copy(copyScratch[acc:], chunkData)
				acc += chunkSize
				rpos = next + chunkSize + 2
			}
			return pos, copyScratch, totalLen, nil
		}

		if int64(totalLen+size) > maxBody {
			return 0, nil, 0, errPayloadTooLarge
		}
		if pos+size+2 > n {
			return 0, nil, 0, errIncompleteRequest
		}
		if data[pos+size] != '\r' || data[pos+size+1] != '\n' {
			return 0, nil, 0, errInvalidRequest
		}

		if firstStart < 0 {
			firstStart = pos
			contiguousEnd = pos + size
		} else {
			switch pos == contiguousEnd {
			case true:
				contiguousEnd = pos + size
			case false:
				contiguousEnd = -1
			}
		}
		totalLen += size
		pos += size + 2
	}
}

func parseChunkSizeLine(data []byte, pos, n int) (size int, next int, err error) {
	if pos >= n {
		return 0, 0, errIncompleteRequest
	}
	hasDigit := false
	i := pos
	for i < n {
		b := data[i]
		if b == '\r' {
			if i+1 >= n {
				return 0, 0, errIncompleteRequest
			}
			if data[i+1] != '\n' {
				return 0, 0, errInvalidRequest
			}
			if !hasDigit {
				return 0, 0, errInvalidRequest
			}
			return size, i + 2, nil
		}
		if b == ';' {
			return 0, 0, errInvalidRequest
		}
		if b >= '0' && b <= '9' {
			hasDigit = true
			d := int(b - '0')
			if size > (1<<60)/16 || size*16 > (1<<60)-d {
				return 0, 0, errInvalidRequest
			}
			size = size*16 + d
			i++
			continue
		}
		if b >= 'a' && b <= 'f' {
			hasDigit = true
			d := int(b - 'a' + 10)
			if size > (1<<60)/16 || size*16 > (1<<60)-d {
				return 0, 0, errInvalidRequest
			}
			size = size*16 + d
			i++
			continue
		}
		if b >= 'A' && b <= 'F' {
			hasDigit = true
			d := int(b - 'A' + 10)
			if size > (1<<60)/16 || size*16 > (1<<60)-d {
				return 0, 0, errInvalidRequest
			}
			size = size*16 + d
			i++
			continue
		}
		if b == ' ' || b == '\t' {
			i++
			continue
		}
		return 0, 0, errInvalidRequest
	}
	return 0, 0, errIncompleteRequest
}

func skipHTTP1ChunkTrailers(data []byte, pos, n int) (int, error) {
	for {
		if pos >= n {
			return 0, errIncompleteRequest
		}
		if data[pos] == '\r' {
			if pos+1 >= n {
				return 0, errIncompleteRequest
			}
			if data[pos+1] != '\n' {
				return 0, errInvalidRequest
			}
			return pos + 2, nil
		}
		for pos < n && data[pos] != '\r' {
			if data[pos] == 0 {
				return 0, errInvalidRequest
			}
			pos++
		}
		if pos+1 >= n {
			return 0, errIncompleteRequest
		}
		if data[pos+1] != '\n' {
			return 0, errInvalidRequest
		}
		pos += 2
	}
}
