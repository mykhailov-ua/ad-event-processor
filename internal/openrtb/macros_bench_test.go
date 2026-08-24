package openrtb

import "testing"

var macroBenchSink []byte

func BenchmarkAppendApplyMacros(b *testing.B) {
	template := []byte("https://win.example/n?price=${AUCTION_PRICE}&id=${AUCTION_ID}&bid=${AUCTION_BID_ID}&imp=${AUCTION_IMP_ID}&seat=${AUCTION_SEAT_ID}")
	ctx := MacroWire{
		AuctionPrice: []byte("1.230000"),
		AuctionID:    []byte("req-1"),
		BidID:        []byte("bid-1"),
		ImpID:        []byte("imp-1"),
		SeatID:       []byte("seat-1"),
	}
	var buf [512]byte
	b.ReportAllocs()
	for b.Loop() {
		macroBenchSink = AppendApplyMacros(buf[:0], template, ctx)
	}
}

func BenchmarkAppendAuctionPrice(b *testing.B) {
	var buf [32]byte
	b.ReportAllocs()
	for b.Loop() {
		macroBenchSink = AppendAuctionPrice(buf[:0], 1_230_000)
	}
}

func BenchmarkGzipCompressInto(b *testing.B) {
	src := make([]byte, 300)
	for i := range src {
		src[i] = 'x'
	}
	dst := make([]byte, 512)
	b.ReportAllocs()
	for b.Loop() {
		_, _ = gzipCompressInto(dst, src)
	}
}
