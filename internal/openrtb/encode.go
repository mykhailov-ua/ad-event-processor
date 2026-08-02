package openrtb

import (
	"encoding/json"
	"math"
)

type BidWire struct {
	RequestID  []byte
	BidID      []byte
	ImpID      []byte
	PriceMicro int64
	CurUSD     bool
	AdM        []byte
	NURL       []byte
	CampaignID []byte
	CreativeID uint64
	DealID     []byte
	SeatID     []byte
}

type BidResponseWire struct {
	RequestID []byte
	BidID     []byte
	CurUSD    bool
	SeatID    []byte
	Bids      []BidWire
}

func AppendBidResponse(dst []byte, p BidWire) ([]byte, error) {
	return AppendBidResponseWire(dst, BidResponseWire{
		RequestID: p.RequestID,
		BidID:     p.BidID,
		CurUSD:    p.CurUSD,
		SeatID:    p.SeatID,
		Bids:      []BidWire{p},
	})
}

func AppendBidResponseWire(dst []byte, w BidResponseWire) ([]byte, error) {
	if len(w.RequestID) == 0 || len(w.BidID) == 0 || len(w.Bids) == 0 {
		return dst, ErrInvalidJSON
	}
	for i := 0; i < len(w.Bids); i++ {
		b := w.Bids[i]
		if len(b.ImpID) == 0 || len(b.CampaignID) == 0 {
			return dst, ErrInvalidJSON
		}
		if len(b.AdM) == 0 && len(b.NURL) == 0 {
			return dst, ErrInvalidJSON
		}
	}
	seat := w.SeatID
	if len(seat) == 0 {
		seat = []byte("1")
	}
	cur := []byte("USD")
	if !w.CurUSD {
		cur = []byte("EUR")
	}

	dst = append(dst, `{"id":`...)
	dst = appendJSONBytes(dst, w.RequestID)
	dst = append(dst, `,"bidid":`...)
	dst = appendJSONBytes(dst, w.BidID)
	dst = append(dst, `,"cur":`...)
	dst = appendJSONBytes(dst, cur)
	dst = append(dst, `,"seatbid":[{"seat":`...)
	dst = appendJSONBytes(dst, seat)
	dst = append(dst, `,"bid":[`...)
	for i := 0; i < len(w.Bids); i++ {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = appendBidObject(dst, w.Bids[i], w.BidID)
	}
	dst = append(dst, `]}]}`...)
	return dst, nil
}

func appendBidObject(dst []byte, p BidWire, defaultBidID []byte) []byte {
	bidID := p.BidID
	if len(bidID) == 0 {
		bidID = defaultBidID
	}
	dst = append(dst, `{"id":`...)
	dst = appendJSONBytes(dst, bidID)
	dst = append(dst, `,"impid":`...)
	dst = appendJSONBytes(dst, p.ImpID)
	dst = append(dst, `,"price":`...)
	dst = appendMicroPrice(dst, p.PriceMicro)
	if len(p.NURL) > 0 {
		dst = append(dst, `,"nurl":`...)
		dst = appendJSONBytes(dst, p.NURL)
	} else {
		dst = append(dst, `,"adm":`...)
		dst = appendJSONBytes(dst, p.AdM)
	}
	dst = append(dst, `,"adid":`...)
	dst = appendJSONBytes(dst, p.CampaignID)
	dst = append(dst, `,"crid":`...)
	dst = append(dst, '"')
	dst = appendUint(dst, p.CreativeID)
	dst = append(dst, '"')
	dst = append(dst, `,"cid":`...)
	dst = appendJSONBytes(dst, p.CampaignID)
	dst = append(dst, `,"adomain":["bidshard.local"]`...)
	if len(p.DealID) > 0 {
		dst = append(dst, `,"dealid":`...)
		dst = appendJSONBytes(dst, p.DealID)
	}
	dst = append(dst, '}')
	return dst
}

func AppendNoBidResponse(dst []byte, requestID []byte, nbr int) []byte {
	dst = append(dst, `{"id":`...)
	if len(requestID) > 0 {
		dst = appendJSONBytes(dst, requestID)
	} else {
		dst = append(dst, `""`...)
	}
	dst = append(dst, `,"nbr":`...)
	dst = appendInt(dst, int64(nbr))
	dst = append(dst, '}')
	return dst
}

func EncodeBid(resp BidResponse) ([]byte, error) {
	return json.Marshal(resp)
}

func EncodeNoBid(requestID string, nbr int) ([]byte, error) {
	return json.Marshal(BidResponse{ID: requestID, NBR: nbr})
}

func MicroToPrice(micro int64) float64 {
	if micro < 0 {
		micro = 0
	}
	return float64(micro) / 1_000_000.0
}

func PriceToMicro(price float64) int64 {
	if price <= 0 || math.IsNaN(price) || math.IsInf(price, 0) {
		return 0
	}
	return int64(price * 1_000_000)
}

func NBRFromReason(reason string) int {
	switch reason {
	case "invalid_request":
		return 2
	case "no_candidates":
		return 1
	case "timeout":
		return 1
	case "deal_mismatch", "pacing_closed":
		return 1
	case "prebid_ivt", "schain_invalid", "breaker_open":
		return 1
	default:
		return 1
	}
}

func appendJSONBytes(dst []byte, b []byte) []byte {
	dst = append(dst, '"')
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch c {
		case '"', '\\':
			dst = append(dst, '\\', c)
		case '\n':
			dst = append(dst, '\\', 'n')
		case '\r':
			dst = append(dst, '\\', 'r')
		case '\t':
			dst = append(dst, '\\', 't')
		default:
			dst = append(dst, c)
		}
	}
	dst = append(dst, '"')
	return dst
}

func appendMicroPrice(dst []byte, micro int64) []byte {
	whole := micro / 1_000_000
	frac := micro % 1_000_000
	if frac < 0 {
		frac = -frac
	}
	if whole < 0 {
		dst = append(dst, '-')
		whole = -whole
	}
	dst = appendUint(dst, uint64(whole))
	dst = append(dst, '.')
	dst = append(dst,
		byte('0'+frac/100000),
	)
	frac %= 100000
	dst = append(dst, byte('0'+frac/10000))
	frac %= 10000
	dst = append(dst, byte('0'+frac/1000))
	frac %= 1000
	dst = append(dst, byte('0'+frac/100))
	frac %= 100
	dst = append(dst, byte('0'+frac/10))
	dst = append(dst, byte('0'+frac%10))
	return dst
}

func appendUint(dst []byte, v uint64) []byte {
	if v == 0 {
		return append(dst, '0')
	}
	var tmp [20]byte
	i := len(tmp)
	for v > 0 {
		i--
		tmp[i] = byte('0' + v%10)
		v /= 10
	}
	return append(dst, tmp[i:]...)
}

func appendInt(dst []byte, v int64) []byte {
	if v == 0 {
		return append(dst, '0')
	}
	if v < 0 {
		dst = append(dst, '-')
		return appendUint(dst, uint64(-v))
	}
	return appendUint(dst, uint64(v))
}
