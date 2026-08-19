package openrtb

const (
	BidHTTPHdrSize          = 129
	bidHTTPContentLenOff    = 89
	bidHTTPContentLenDigits = 12

	bidHTTPJSONReserve = 192
)

var (
	bidHTTPHdrPrefix  = []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nx-openrtb-version: 2.6\r\nContent-Length: ")
	bidHTTPHdrSuffix  = []byte("\r\nConnection: keep-alive\r\n\r\n")
	bidHTTPGzipPrefix = []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nx-openrtb-version: 2.6\r\nContent-Encoding: gzip\r\nContent-Length: ")
)

const BidJSONHdrReserve = BidHTTPHdrSize

func WriteBidHTTPResponse(buf []byte, p BidWire, opts HTTPWriteOpts) (int, error) {
	return WriteBidsHTTPResponse(buf, BidResponseWire{
		RequestID: p.RequestID,
		BidID:     p.BidID,
		CurUSD:    p.CurUSD,
		SeatID:    p.SeatID,
		Bids:      []BidWire{p},
	}, opts)
}

func WriteBidsHTTPResponse(buf []byte, w BidResponseWire, opts HTTPWriteOpts) (int, error) {
	if !opts.Gzip {
		if len(buf) < BidHTTPHdrSize+32 {
			return 0, ErrBodyTooLarge
		}
		jsonEnd, err := AppendBidResponseWire(buf[BidHTTPHdrSize:BidHTTPHdrSize], w)
		if err != nil {
			return 0, err
		}
		jsonLen := len(jsonEnd) - BidHTTPHdrSize
		writeBid200Header(buf, false)
		patchContentLength12(buf, bidHTTPContentLenOff, jsonLen)
		return BidHTTPHdrSize + jsonLen, nil
	}
	if len(buf) < bidHTTPJSONReserve+32 {
		return 0, ErrBodyTooLarge
	}
	jsonStart := bidHTTPJSONReserve
	jsonEnd, err := AppendBidResponseWire(buf[jsonStart:jsonStart], w)
	if err != nil {
		return 0, err
	}
	return writeHTTP200JSONGzip(buf, jsonEnd)
}

func WriteNoBidHTTPResponse(buf, requestID []byte, nbr int, opts HTTPWriteOpts) (int, error) {
	if !opts.Gzip {
		if len(buf) < BidHTTPHdrSize+16 {
			return 0, ErrBodyTooLarge
		}
		jsonEnd := AppendNoBidResponse(buf[BidHTTPHdrSize:BidHTTPHdrSize], requestID, nbr)
		jsonLen := len(jsonEnd) - BidHTTPHdrSize
		writeBid200Header(buf, false)
		patchContentLength12(buf, bidHTTPContentLenOff, jsonLen)
		return BidHTTPHdrSize + jsonLen, nil
	}
	if len(buf) < bidHTTPJSONReserve+16 {
		return 0, ErrBodyTooLarge
	}
	jsonStart := bidHTTPJSONReserve
	jsonEnd := AppendNoBidResponse(buf[jsonStart:jsonStart], requestID, nbr)
	return writeHTTP200JSONGzip(buf, jsonEnd)
}

func writeHTTP200JSON(buf, jsonBody []byte, opts HTTPWriteOpts) (int, error) {
	if !shouldGzipBody(len(jsonBody), opts) {
		if len(buf) < BidHTTPHdrSize+len(jsonBody) {
			return 0, ErrBodyTooLarge
		}
		copy(buf[BidHTTPHdrSize:], jsonBody)
		writeBid200Header(buf, false)
		patchContentLength12(buf, bidHTTPContentLenOff, len(jsonBody))
		return BidHTTPHdrSize + len(jsonBody), nil
	}
	return writeHTTP200JSONGzip(buf, jsonBody)
}

func writeHTTP200JSONGzip(buf, jsonBody []byte) (int, error) {
	hdrLen := len(bidHTTPGzipPrefix) + bidHTTPContentLenDigits + len(bidHTTPHdrSuffix)
	if len(buf) < hdrLen+len(jsonBody) {
		return 0, ErrBodyTooLarge
	}
	bodyOff := hdrLen
	compLen, err := gzipCompressInto(buf[bodyOff:], jsonBody)
	if err != nil {
		return 0, err
	}
	writeBid200Header(buf, true)
	patchContentLength12(buf, len(bidHTTPGzipPrefix), compLen)
	return bodyOff + compLen, nil
}

func WriteJSONHTTPResponse(buf, body []byte, opts HTTPWriteOpts) (int, error) {
	return writeHTTP200JSON(buf, body, opts)
}

func writeBid200Header(buf []byte, gzip bool) {
	var prefix []byte
	if gzip {
		prefix = bidHTTPGzipPrefix
	} else {
		prefix = bidHTTPHdrPrefix
	}
	n := copy(buf, prefix)
	for i := range bidHTTPContentLenDigits {
		buf[n+i] = '0'
	}
	copy(buf[n+bidHTTPContentLenDigits:], bidHTTPHdrSuffix)
}

func patchContentLength12(buf []byte, off, n int) {
	for i := bidHTTPContentLenDigits - 1; i >= 0; i-- {
		buf[off+i] = byte('0' + n%10)
		n /= 10
	}
}
