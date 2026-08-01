package openrtb

import "testing"

func BenchmarkAppendBidResponse(b *testing.B) {
	var reqID, bidID, impID, camp [36]byte
	copy(reqID[:], "req-golden-001")
	copy(bidID[:], "bid-abc")
	copy(impID[:], "imp-1")
	copy(camp[:], "camp-uuid-0001-0002-0003-0004")
	adm := []byte(`<html><body>ad</body></html>`)
	p := BidWire{
		RequestID:  reqID[:len("req-golden-001")],
		BidID:      bidID[:len("bid-abc")],
		ImpID:      impID[:len("imp-1")],
		PriceMicro: 2_000_000,
		CurUSD:     true,
		AdM:        adm,
		CampaignID: camp[:len("camp-uuid-0001-0002-0003-0004")],
		CreativeID: 7,
	}
	var buf [2048]byte
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = AppendBidResponse(buf[:0], p)
	}
}

func BenchmarkWriteBidHTTPResponse(b *testing.B) {
	var reqID, bidID, impID, camp [36]byte
	copy(reqID[:], "req-golden-001")
	copy(bidID[:], "bid-abc")
	copy(impID[:], "imp-1")
	copy(camp[:], "camp-uuid-0001-0002-0003-0004")
	adm := []byte(`<html><body>ad</body></html>`)
	p := BidWire{
		RequestID:  reqID[:len("req-golden-001")],
		BidID:      bidID[:len("bid-abc")],
		ImpID:      impID[:len("imp-1")],
		PriceMicro: 2_000_000,
		CurUSD:     true,
		AdM:        adm,
		CampaignID: camp[:len("camp-uuid-0001-0002-0003-0004")],
		CreativeID: 7,
	}
	var buf [2048]byte
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = WriteBidHTTPResponse(buf[:], p, HTTPWriteOpts{})
	}
}
