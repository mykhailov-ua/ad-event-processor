package ingestion

import (
	"bytes"
	"fmt"
	"testing"
)

var http1HappyCorpus = []byte(
	"POST /track HTTP/1.1\r\n" +
		"Content-Length: 69\r\n" +
		"\r\n" +
		`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}`,
)

var http1OpenRTBBidCorpus = func() []byte {
	body := []byte(`{"id":"req-1","imp":[{"id":"1","banner":{"w":300,"h":250}}]}`)
	var wire []byte
	wire = append(wire, "POST /openrtb/bid HTTP/1.1\r\n"...)
	wire = append(wire, "Host: edge.local\r\n"...)
	wire = append(wire, "Content-Type: application/json\r\n"...)
	wire = append(wire, fmt.Sprintf("Content-Length: %d\r\n", len(body))...)
	wire = append(wire, "X-Forwarded-For: 203.0.113.10\r\n"...)
	wire = append(wire, "User-Agent: Mozilla/5.0\r\n\r\n"...)
	wire = append(wire, body...)
	return wire
}()

var http1WorstCorpus = nginxTrackCorpus

func BenchmarkHTTP1DFA_Happy(b *testing.B) {
	const maxBody = int64(1024 * 1024)
	b.SetBytes(int64(len(http1HappyCorpus)))
	b.ReportAllocs()
	for b.Loop() {
		_, _, err := parseHTTP1(http1HappyCorpus, maxBody, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHTTP1DFA_OpenRTBBid(b *testing.B) {
	const maxBody = int64(1024 * 1024)
	b.SetBytes(int64(len(http1OpenRTBBidCorpus)))
	b.ReportAllocs()
	for b.Loop() {
		_, _, err := parseHTTP1(http1OpenRTBBidCorpus, maxBody, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHTTP1DFA_Worst(b *testing.B) {
	const maxBody = int64(1024 * 1024)
	b.SetBytes(int64(len(http1WorstCorpus)))
	b.ReportAllocs()
	for b.Loop() {
		_, _, err := parseHTTP1(http1WorstCorpus, maxBody, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHTTP2DFA_Happy(b *testing.B) {
	buf := []byte{0x00, 0x00, 0x05, h2FrameData, 0x00, 0x00, 0x00, 0x00, 0x01, 'h', 'e', 'l', 'l', 'o'}
	b.SetBytes(int64(len(buf)))
	b.ReportAllocs()
	for b.Loop() {
		_, _, err := decodeH2FrameHeader(buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHTTP2DFA_Worst(b *testing.B) {
	body := []byte(`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}`)
	wire := buildH2TrackRequest(body)
	b.SetBytes(int64(len(wire)))
	b.ReportAllocs()
	st := newH2ConnState()
	for b.Loop() {
		st.resetConn()
		_, _, _, _, err := parseH2Ingress(wire, &st, 1<<20)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHTTP3DFA_Happy(b *testing.B) {
	buf := []byte{0x25}
	b.SetBytes(int64(len(buf)))
	b.ReportAllocs()
	for b.Loop() {
		_, _, err := quicDecodeVarint(buf, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHTTP3DFA_Worst(b *testing.B) {
	body := []byte(`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}`)
	var hdrBlock []byte
	hdrBlock = append(hdrBlock, 0x83, 0x04, 0x06, '/', 't', 'r', 'a', 'c', 'k')
	var buf bytes.Buffer
	writeHTTP3Frame(&buf, h3FrameHeaders, hdrBlock)
	writeHTTP3Frame(&buf, h3FrameData, body)
	wire := buf.Bytes()
	b.SetBytes(int64(len(wire)))
	b.ReportAllocs()
	for b.Loop() {
		_, _, err := h3ParseRequestFrames(wire, 1<<20)
		if err != nil {
			b.Fatal(err)
		}
	}
}
