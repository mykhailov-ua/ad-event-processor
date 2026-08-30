package httpingress

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
)

type IngressVerdict string

const (
	IngressAccept     IngressVerdict = "accept"
	IngressReject     IngressVerdict = "reject"
	IngressIncomplete IngressVerdict = "incomplete"
)

type Disposition struct {
	Verdict  IngressVerdict
	BodyLen  int
	Consumed int
}

func DispositionFromHTTP1Parse(n int, req Request, err error) Disposition {
	if err != nil {
		if errors.Is(err, ErrIncomplete) {
			return Disposition{Verdict: IngressIncomplete}
		}
		return Disposition{Verdict: IngressReject}
	}
	bodyLen := req.ContentLength
	if req.Body != nil {
		bodyLen = len(req.Body)
	}
	return Disposition{
		Verdict:  IngressAccept,
		BodyLen:  bodyLen,
		Consumed: n,
	}
}

func EdgeHTTP1Disposition(wire []byte, maxBody int64) Disposition {
	return GnetHTTP1Disposition(wire, maxBody)
}

func GnetHTTP1Disposition(wire []byte, maxBody int64) Disposition {
	n, req, err := ParseHTTP1(wire, maxBody, nil)
	return DispositionFromHTTP1Parse(n, req, err)
}

func HTTP1IngressCanonical(wire []byte, maxBody int64) (edge, gnet Disposition, differential bool) {
	edge = EdgeHTTP1Disposition(wire, maxBody)
	gnet = GnetHTTP1Disposition(wire, maxBody)
	differential = edge.Verdict != gnet.Verdict ||
		(edge.Verdict == IngressAccept && edge.BodyLen != gnet.BodyLen) ||
		(edge.Verdict == IngressAccept && edge.Consumed != gnet.Consumed)
	return edge, gnet, differential
}

type CorpusCase struct {
	Name    string
	Wire    []byte
	MaxBody int64
}

func BuildNginxIngressCorpus() []CorpusCase {
	const edgeMax = int64(1024 * 1024)
	const faultMax = int64(1024)

	out := make([]CorpusCase, 0, 280)

	add := func(name string, wire []byte, maxBody int64) {
		out = append(out, CorpusCase{Name: name, Wire: wire, MaxBody: maxBody})
	}

	for _, tc := range FraudHTTP1Cases2026() {
		add("fraud_"+tc.ID, tc.Payload, tc.MaxBody)
	}
	for _, tc := range HTTP1FaultMalformedCases() {
		add("fault_"+tc.Name, tc.Payload, tc.MaxBody)
	}
	add("nginx_track_corpus", NginxTrackCorpus, edgeMax)

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

const (
	swarByteHi  = 0x8080808080808080
	swarTokenLo = 0x2121212121212121
	swarTokenHi = 0x7E7E7E7E7E7E7E7E
	swarHOne    = 0x0101010101010101
)

var httpHeaderValOK [256]byte

func initHTTP1ValidateTables() {
	for i := 0x20; i <= 0x7E; i++ {
		httpHeaderValOK[i] = 1
	}
	httpHeaderValOK['\t'] = 1
}

func swarHasHighBit(x uint64) bool {
	return x&swarByteHi != 0
}

func HTTPTokenValid(b []byte) bool {
	return httpTokenValid(b)
}

func httpTokenValid(b []byte) bool {
	n := len(b)
	if n == 0 {
		return false
	}
	_ = b[n-1]
	i := 0
	for i+8 <= n {
		v := binary.LittleEndian.Uint64(b[i:])
		t := v - swarTokenLo
		if swarHasHighBit(t) {
			return false
		}
		t = swarTokenHi - v
		if swarHasHighBit(t) {
			return false
		}
		i += 8
	}
	for i < n {
		c := b[i]
		if c < 0x21 || c > 0x7E {
			return false
		}
		i++
	}
	return true
}

func HTTPHeaderValValid(b []byte) bool {
	return httpHeaderValValid(b)
}

func httpHeaderValValid(b []byte) bool {
	n := len(b)
	if n == 0 {
		return true
	}
	_ = b[n-1]
	i := 0
	for i+8 <= n {
		j := i
		if httpHeaderValOK[b[j]] == 0 ||
			httpHeaderValOK[b[j+1]] == 0 ||
			httpHeaderValOK[b[j+2]] == 0 ||
			httpHeaderValOK[b[j+3]] == 0 ||
			httpHeaderValOK[b[j+4]] == 0 ||
			httpHeaderValOK[b[j+5]] == 0 ||
			httpHeaderValOK[b[j+6]] == 0 ||
			httpHeaderValOK[b[j+7]] == 0 {
			return false
		}
		i += 8
	}
	for i < n {
		if httpHeaderValOK[b[i]] == 0 {
			return false
		}
		i++
	}
	return true
}

func http1KeyMatchFold(key []byte, lit string) bool {
	if len(key) != len(lit) {
		return false
	}
	for i := range len(lit) {
		if httpFold[key[i]] != lit[i] {
			return false
		}
	}
	return true
}

const (
	wireSecFetchSiteBit uint8 = 1 << 0
	wireSecFetchModeBit uint8 = 1 << 1
	wireSecFetchDestBit uint8 = 1 << 2
)

var assignConnTimingHeadersFn func(req *Request, key, val []byte)
