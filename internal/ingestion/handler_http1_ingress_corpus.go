package ingestion

import (
	"bytes"
	"fmt"
	"strconv"
)

type ingressCorpusCase struct {
	Name    string
	Wire    []byte
	MaxBody int64
}

func buildNginxIngressCorpus() []ingressCorpusCase {
	const edgeMax = int64(1024 * 1024)
	const faultMax = int64(1024)

	out := make([]ingressCorpusCase, 0, 280)

	add := func(name string, wire []byte, maxBody int64) {
		out = append(out, ingressCorpusCase{Name: name, Wire: wire, MaxBody: maxBody})
	}

	for _, tc := range fraudHTTP1Cases2026() {
		add("fraud_"+tc.id, tc.payload, tc.maxBody)
	}
	for _, tc := range http1FaultMalformedCases() {
		add("fault_"+tc.name, tc.payload, tc.maxBody)
	}
	add("nginx_track_corpus", nginxTrackCorpus, edgeMax)

	minimalPOST := []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n")
	add("minimal_post_track", minimalPOST, faultMax)

	for n := 0; n <= 64; n++ {
		body := bytes.Repeat([]byte("a"), n)
		hdr := fmt.Sprintf("POST /track HTTP/1.1\r\nContent-Length: %d\r\n\r\n", n)
		add(fmt.Sprintf("track_cl_body_%d", n), append([]byte(hdr), body...), faultMax)
	}

	for piped := 2; piped <= 20; piped++ {
		add(fmt.Sprintf("pipeline_%d", piped), bytes.Repeat(minimalPOST, piped), faultMax)
	}

	xffVariants := []string{
		"203.0.113.1",
		"::1, 203.0.113.1",
		"10.0.0.1, 192.0.2.1, 203.0.113.5",
		"198.51.100.2, 203.0.113.1, 10.0.0.1, 192.0.2.1",
	}
	for i, xff := range xffVariants {
		wire := []byte("POST /track HTTP/1.1\r\nX-Forwarded-For: " + xff + "\r\nContent-Length: 0\r\n\r\n")
		add(fmt.Sprintf("xff_chain_%d", i), wire, faultMax)
	}

	teVectors := [][]byte{
		[]byte("POST /track HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n"),
		[]byte("POST /track HTTP/1.1\r\nTransfer-Encoding: chunked\r\nContent-Length: 0\r\n\r\n"),
		[]byte("POST /track HTTP/1.1\r\nTransfer-Encoding: gzip\r\nContent-Length: 5\r\n\r\nhello"),
		[]byte("POST /track HTTP/1.1\r\nTransfer-Encoding: chunked, gzip\r\nContent-Length: 0\r\n\r\n"),
	}
	for i, wire := range teVectors {
		add(fmt.Sprintf("te_vector_%d", i), wire, faultMax)
	}

	openrtbChunked := append([]byte(
		"POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding: chunked\r\nContent-Type: application/json\r\n\r\n"),
		[]byte("5\r\n")...)
	openrtbChunked = append(openrtbChunked, `{"id":1}`...)
	openrtbChunked = append(openrtbChunked, "\r\n0\r\n\r\n"...)
	add("openrtb_chunked_ok", openrtbChunked, edgeMax)

	paths := []string{"/track", "/track?cb=1", "/openrtb/bid", "/health", "/ready", "/metrics"}
	for _, path := range paths {
		wire := []byte("GET " + path + " HTTP/1.1\r\nContent-Length: 0\r\n\r\n")
		add("get_"+path, wire, faultMax)
	}

	for i := range 32 {
		cl := strconv.Itoa(i % 17)
		wire := []byte("POST /track HTTP/1.1\r\nContent-Length:\t" + cl + "\r\n\r\n")
		if i%17 > 0 {
			wire = append(wire, bytes.Repeat([]byte("x"), i%17)...)
		}
		add(fmt.Sprintf("cl_tab_variant_%d", i), wire, faultMax)
	}

	for i := range 32 {
		pad := bytes.Repeat([]byte("h"), i%8)
		wire := append([]byte("POST /track HTTP/1.1\r\nX-Pad: "), pad...)
		wire = append(wire, []byte("\r\nContent-Length: 0\r\n\r\n")...)
		add(fmt.Sprintf("header_pad_%d", i), wire, faultMax)
	}

	for i := range 24 {
		wire := append([]byte("POST /track HTTP/1.1\r\nContent-Length: "), []byte(strconv.Itoa(1024+i))...)
		wire = append(wire, "\r\n\r\n"...)
		wire = append(wire, bytes.Repeat([]byte("z"), 1024+i)...)
		add(fmt.Sprintf("near_max_body_%d", i), wire, faultMax)
	}

	smuggleTails := []string{"", "X", "SMUGGLED", "GET /admin HTTP/1.1\r\n\r\n"}
	for i, tail := range smuggleTails {
		wire := []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n" + tail)
		add(fmt.Sprintf("cl_zero_tail_%d", i), wire, faultMax)
	}

	if len(out) < 200 {
		panic(fmt.Sprintf("nginx ingress corpus too small: %d", len(out)))
	}
	return out
}
