package ingestion

import "bytes"

type http1FaultCase struct {
	name    string
	payload []byte
	maxBody int64
	wantOK  bool
	wantErr error
}

func http1FaultMalformedCases() []http1FaultCase {
	const maxBody = int64(1024)
	return []http1FaultCase{
		{name: "empty", payload: nil, maxBody: maxBody, wantErr: errIncompleteRequest},
		{name: "single_null", payload: []byte{0}, maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "binary_garbage", payload: randomWireGarbage(256), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "null_in_request_line", payload: []byte("POS\x00T /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "null_in_header_name", payload: []byte("POST /track HTTP/1.1\r\nX-Fo\x00o: bar\r\nContent-Length: 0\r\n\r\n"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "null_in_header_value", payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\nX-Test: a\x00b\r\n\r\n"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "lf_only_line_endings", payload: []byte("POST /track HTTP/1.1\nContent-Length: 0\n\n"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "cr_only_no_lf", payload: []byte("POST /track HTTP/1.1\rContent-Length: 0\r\r"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "header_line_no_colon", payload: []byte("POST /track HTTP/1.1\r\nBadHeader\r\n\r\n"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "colon_no_value", payload: []byte("POST /track HTTP/1.1\r\nContent-Length:\r\n\r\n"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "content_length_alpha", payload: []byte("POST /track HTTP/1.1\r\nContent-Length: abc\r\n\r\n"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "content_length_mixed", payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 12abc\r\n\r\nhello"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "content_length_huge", payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 999999999999999999\r\n\r\n"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "content_length_at_limit_plus_one", payload: append([]byte("POST /track HTTP/1.1\r\nContent-Length: 1025\r\n\r\n"), bytes.Repeat([]byte("x"), 1025)...), maxBody: maxBody, wantErr: errPayloadTooLarge},
		{name: "content_length_at_limit", payload: append([]byte("POST /track HTTP/1.1\r\nContent-Length: 1024\r\n\r\n"), bytes.Repeat([]byte("y"), 1024)...), maxBody: maxBody, wantOK: true},
		{name: "body_shorter_than_cl", payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 100\r\n\r\nshort"), maxBody: maxBody, wantErr: errIncompleteRequest},
		{name: "body_longer_than_cl", payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 3\r\n\r\nabcdef"), maxBody: maxBody, wantOK: true},
		{name: "double_crlf_early", payload: []byte("POST /track HTTP/1.1\r\n\r\njunk"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "http_09_style", payload: []byte("GET /track\r\n\r\n"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "method_overflow", payload: append(append([]byte(nil), bytes.Repeat([]byte("A"), 8192)...), []byte(" /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n")...), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "path_null_byte", payload: []byte("POST /tra\x00ck HTTP/1.1\r\nContent-Length: 0\r\n\r\n"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "fast_path_corrupt_version", payload: []byte("POST /track HTTP/2.0\r\nContent-Length: 0\r\n\r\n"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "fast_path_corrupt_method", payload: []byte("POST /track HTTP/1.1\r\nHost: x\r\nContent-Length: 0\r\n\r\n"), maxBody: maxBody, wantOK: true},
		{name: "utf8_header_value", payload: []byte("POST /track HTTP/1.1\r\nUser-Agent: тест\xFF\xFE\r\nContent-Length: 0\r\n\r\n"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "crlf_in_header_value", payload: []byte("POST /track HTTP/1.1\r\nX-Evil: foo\r\nbar\r\nContent-Length: 0\r\n\r\n"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "negative_looking_cl", payload: []byte("POST /track HTTP/1.1\r\nContent-Length: -1\r\n\r\n"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "leading_zero_cl", payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 0005\r\n\r\nhello"), maxBody: maxBody, wantOK: true},
		{name: "pipelined_valid_then_garbage", payload: append(append([]byte(nil), nginxTrackCorpus...), randomWireGarbage(64)...), maxBody: 1024 * 1024, wantOK: true},
		{name: "chunked_track_rejected", payload: []byte("POST /track HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n"), maxBody: maxBody, wantErr: errInvalidRequest},
		{name: "chunked_openrtb_ok", payload: []byte("POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n"), maxBody: maxBody, wantOK: true},
		{name: "control_chars_in_method", payload: []byte("PO\x01ST /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n"), maxBody: maxBody, wantErr: errInvalidRequest},
	}
}

func randomWireGarbage(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}
