package openrtb

import (
	"testing"
)

var fuzzBidRequestMin = []byte(`{
 "id": "req-golden-001",
 "tmax": 250,
 "cur": ["USD"],
 "imp": [{
 "id": "imp-1",
 "bidfloor": 1.25,
 "banner": {"w": 300, "h": 250}
 }],
 "site": {"domain": "example.com", "page": "https://example.com/"},
 "device": {"ip": "203.0.113.1", "ua": "Mozilla/5.0", "devicetype": 2, "geo": {"country": "USA"}}
}`)

func fuzzNoPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s: panic: %v", name, r)
		}
	}()
	fn()
}

func FuzzValidateBytes(f *testing.F) {
	f.Add(fuzzBidRequestMin)
	f.Add([]byte(`{"openrtb":{"request":{}}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`not json`))

	cfg := ExchangeConfig{MultiImpMax: 10}
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzNoPanic(t, "ValidateBytes", func() {
			_ = ValidateBytes(data, cfg)
		})
		fuzzNoPanic(t, "ExchangeBodyPrecheck", func() {
			_ = ExchangeBodyPrecheck(data)
		})
		fuzzNoPanic(t, "Decode", func() {
			_, _ = Decode(data)
		})
		fuzzNoPanic(t, "IsOpenRTB30Shape", func() {
			_ = IsOpenRTB30Shape(data)
		})
	})
}

func FuzzAppendApplyMacros(f *testing.F) {
	f.Add([]byte(`https://track.example/n?price=${AUCTION_PRICE}&id=${AUCTION_ID}`), []byte("1.25"), []byte("req-1"), []byte("bid-1"), []byte("imp-1"), []byte("seat-1"))
	f.Add([]byte("no macros"), []byte(""), []byte(""), []byte(""), []byte(""), []byte(""))

	f.Fuzz(func(t *testing.T, template, price, auctionID, bidID, impID, seatID []byte) {
		var dst [1024]byte
		ctx := MacroWire{
			AuctionPrice: price,
			AuctionID:    auctionID,
			BidID:        bidID,
			ImpID:        impID,
			SeatID:       seatID,
		}
		fuzzNoPanic(t, "AppendApplyMacros", func() {
			_ = AppendApplyMacros(dst[:0], template, ctx)
		})
		fuzzNoPanic(t, "AppendAuctionPrice", func() {
			_ = AppendAuctionPrice(dst[:0], int64(len(price))*1000)
		})
		fuzzNoPanic(t, "AppendCreativeID", func() {
			_ = AppendCreativeID(dst[:0], uint64(len(template)))
		})
	})
}

func FuzzGzipCompress(f *testing.F) {
	f.Add(fuzzBidRequestMin)
	f.Add([]byte("short"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		var dst [4096]byte
		fuzzNoPanic(t, "gzipCompressInto", func() {
			_, _ = gzipCompressInto(dst[:], data)
		})
		fuzzNoPanic(t, "shouldGzipBody", func() {
			_ = shouldGzipBody(len(data), HTTPWriteOpts{Gzip: true})
		})
	})
}
