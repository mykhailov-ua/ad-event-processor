package httpingress

import (
	"bytes"
)

type HTTP1FaultCase struct {
	Name    string
	Payload []byte
	MaxBody int64
	WantOK  bool
	WantErr error
}

func HTTP1FaultMalformedCases() []HTTP1FaultCase {
	const maxBody = int64(1024)
	return []HTTP1FaultCase{
		{Name: "empty", Payload: nil, MaxBody: maxBody, WantErr: ErrIncomplete},
		{Name: "single_null", Payload: []byte{0}, MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "binary_garbage", Payload: RandomWireGarbage(256), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "null_in_request_line", Payload: []byte("POS\x00T /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "null_in_header_name", Payload: []byte("POST /track HTTP/1.1\r\nX-Fo\x00o: bar\r\nContent-Length: 0\r\n\r\n"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "null_in_header_value", Payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\nX-Test: a\x00b\r\n\r\n"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "lf_only_line_endings", Payload: []byte("POST /track HTTP/1.1\nContent-Length: 0\n\n"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "cr_only_no_lf", Payload: []byte("POST /track HTTP/1.1\rContent-Length: 0\r\r"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "header_line_no_colon", Payload: []byte("POST /track HTTP/1.1\r\nBadHeader\r\n\r\n"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "colon_no_value", Payload: []byte("POST /track HTTP/1.1\r\nContent-Length:\r\n\r\n"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "content_length_alpha", Payload: []byte("POST /track HTTP/1.1\r\nContent-Length: abc\r\n\r\n"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "content_length_mixed", Payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 12abc\r\n\r\nhello"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "content_length_huge", Payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 999999999999999999\r\n\r\n"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "content_length_at_limit_plus_one", Payload: append([]byte("POST /track HTTP/1.1\r\nContent-Length: 1025\r\n\r\n"), bytes.Repeat([]byte("x"), 1025)...), MaxBody: maxBody, WantErr: ErrPayloadTooLarge},
		{Name: "content_length_at_limit", Payload: append([]byte("POST /track HTTP/1.1\r\nContent-Length: 1024\r\n\r\n"), bytes.Repeat([]byte("y"), 1024)...), MaxBody: maxBody, WantOK: true},
		{Name: "body_shorter_than_cl", Payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 100\r\n\r\nshort"), MaxBody: maxBody, WantErr: ErrIncomplete},
		{Name: "body_longer_than_cl", Payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 3\r\n\r\nabcdef"), MaxBody: maxBody, WantOK: true},
		{Name: "double_crlf_early", Payload: []byte("POST /track HTTP/1.1\r\n\r\njunk"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "http_09_style", Payload: []byte("GET /track\r\n\r\n"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "method_overflow", Payload: append(append([]byte(nil), bytes.Repeat([]byte("A"), 8192)...), []byte(" /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n")...), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "path_null_byte", Payload: []byte("POST /tra\x00ck HTTP/1.1\r\nContent-Length: 0\r\n\r\n"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "fast_path_corrupt_version", Payload: []byte("POST /track HTTP/2.0\r\nContent-Length: 0\r\n\r\n"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "fast_path_corrupt_method", Payload: []byte("POST /track HTTP/1.1\r\nHost: x\r\nContent-Length: 0\r\n\r\n"), MaxBody: maxBody, WantOK: true},
		{Name: "utf8_header_value", Payload: []byte("POST /track HTTP/1.1\r\nUser-Agent: test\xFF\xFE\r\nContent-Length: 0\r\n\r\n"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "crlf_in_header_value", Payload: []byte("POST /track HTTP/1.1\r\nX-Evil: foo\r\nbar\r\nContent-Length: 0\r\n\r\n"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "negative_looking_cl", Payload: []byte("POST /track HTTP/1.1\r\nContent-Length: -1\r\n\r\n"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "leading_zero_cl", Payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 0005\r\n\r\nhello"), MaxBody: maxBody, WantOK: true},
		{Name: "pipelined_valid_then_garbage", Payload: append(append([]byte(nil), NginxTrackCorpus...), RandomWireGarbage(64)...), MaxBody: 1024 * 1024, WantOK: true},
		{Name: "chunked_track_rejected", Payload: []byte("POST /track HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n"), MaxBody: maxBody, WantErr: ErrInvalid},
		{Name: "chunked_openrtb_ok", Payload: []byte("POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n"), MaxBody: maxBody, WantOK: true},
		{Name: "control_chars_in_method", Payload: []byte("PO\x01ST /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n"), MaxBody: maxBody, WantErr: ErrInvalid},
	}
}

func RandomWireGarbage(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}

type FraudHTTP1Case struct {
	ID      string
	Name    string
	Payload []byte
	MaxBody int64
	MustErr bool
	WantErr error
	WantOK  bool
}

func FraudHTTP1Cases2026() []FraudHTTP1Case {
	const maxBody = int64(1024)
	minimalPOST := []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n")

	return []FraudHTTP1Case{
		{
			ID: "H1-01", Name: "cl_te_smuggling_gzip_chunked",
			Payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 6\r\nTransfer-Encoding: gzip, chunked\r\n\r\n0\r\n\r\n"),
			MaxBody: maxBody, MustErr: true, WantErr: ErrInvalid,
		},
		{
			ID: "H1-02", Name: "duplicate_cl_desync",
			Payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\nContent-Length: 5\r\n\r\nhello"),
			MaxBody: maxBody, MustErr: true, WantErr: ErrInvalid,
		},
		{
			ID: "H1-03", Name: "cl_zero_smuggled_tail",
			Payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\n\r\nSMUGGLED"),
			MaxBody: maxBody, WantOK: true,
		},
		{
			ID: "H1-04", Name: "obs_fold_header_injection",
			Payload: []byte("POST /track HTTP/1.1\r\nX-Forwarded-For: 1.2.3.4\r\n injected: evil\r\nContent-Length: 0\r\n\r\n"),
			MaxBody: maxBody, MustErr: true, WantErr: ErrInvalid,
		},
		{
			ID: "H1-05", Name: "tls_hash_oversize_header",
			Payload: append(append([]byte("POST /track HTTP/1.1\r\nX-TLS-Hash: "), bytes.Repeat([]byte("a"), 10240)...), []byte("\r\nContent-Length: 0\r\n\r\n")...),
			MaxBody: maxBody, MustErr: true, WantErr: ErrInvalid,
		},
		{
			ID: "H1-06", Name: "sec_ch_ua_ua_mismatch_still_parses",
			Payload: []byte("POST /track HTTP/1.1\r\nUser-Agent: Chrome/120\r\nSec-CH-UA: \"Safari\";v=\"17\"\r\nContent-Length: 0\r\n\r\n"),
			MaxBody: maxBody, WantOK: true,
		},
		{
			ID: "H1-07", Name: "click_spam_pipelined_50",
			Payload: bytes.Repeat(minimalPOST, 50),
			MaxBody: maxBody, WantOK: true,
		},
		{
			ID: "H1-09", Name: "x_original_method_ignored",
			Payload: []byte("POST /track HTTP/1.1\r\nX-Original-Method: GET\r\nContent-Length: 0\r\n\r\n"),
			MaxBody: maxBody, WantOK: true,
		},
		{
			ID: "H1-10", Name: "http_10_downgrade",
			Payload: []byte("POST /track HTTP/1.0\r\nContent-Length: 0\r\n\r\n"),
			MaxBody: maxBody, MustErr: true, WantErr: ErrInvalid,
		},
		{
			ID: "H1-11", Name: "homoglyph_cyrillic_track_path",
			Payload: []byte("POST /tr\u0430ck HTTP/1.1\r\nContent-Length: 0\r\n\r\n"),
			MaxBody: maxBody, MustErr: true, WantErr: ErrInvalid,
		},
		{
			ID: "H1-12", Name: "xff_long_chain",
			Payload: []byte("POST /track HTTP/1.1\r\nX-Forwarded-For: ::1, 203.0.113.1, 10.0.0.1, 192.0.2.1\r\nContent-Length: 0\r\n\r\n"),
			MaxBody: maxBody, WantOK: true,
		},
		{
			ID: "H1-13", Name: "te_chunked_with_cl_zero",
			Payload: []byte("POST /track HTTP/1.1\r\nTransfer-Encoding: chunked\r\nContent-Length: 0\r\n\r\n"),
			MaxBody: maxBody, MustErr: true, WantErr: ErrInvalid,
		},
		{
			ID: "H1-14", Name: "cl_tab_prefix",
			Payload: []byte("POST /track HTTP/1.1\r\nContent-Length:\t5\r\n\r\nhello"),
			MaxBody: maxBody, WantOK: true,
		},
	}
}

const (
	http1MaxMethodLen     = 16
	http1MaxPathLen       = 2048
	http1MaxHeaderNameLen = 256
	http1MaxHeaderValLen  = 1024
)

var (
	httpFold [256]byte

	trackReqLine = [22]byte{
		'P', 'O', 'S', 'T', ' ', '/', 't', 'r', 'a', 'c', 'k', ' ',
		'H', 'T', 'T', 'P', '/', '1', '.', '1', '\r', '\n',
	}
	openrtbBidReqLine = [29]byte{
		'P', 'O', 'S', 'T', ' ', '/', 'o', 'p', 'e', 'n', 'r', 't', 'b', '/', 'b', 'i', 'd', ' ',
		'H', 'T', 'T', 'P', '/', '1', '.', '1', '\r', '\n',
	}
)

func init() {
	initHTTP1ValidateTables()
	for i := range 256 {
		httpFold[i] = byte(i)
	}
	for i := 'A'; i <= 'Z'; i++ {
		httpFold[i] = byte(i + ('a' - 'A'))
	}
}
