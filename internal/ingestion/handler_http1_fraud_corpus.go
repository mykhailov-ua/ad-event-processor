package ingestion

import "bytes"

type fraudHTTP1Case struct {
	id      string
	name    string
	payload []byte
	maxBody int64
	mustErr bool
	wantErr error
	mustOK  bool
}

func fraudHTTP1Cases2026() []fraudHTTP1Case {
	const maxBody = int64(1024)
	minimalPOST := []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n")

	return []fraudHTTP1Case{
		{
			id: "H1-01", name: "cl_te_smuggling_gzip_chunked",
			payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 6\r\nTransfer-Encoding: gzip, chunked\r\n\r\n0\r\n\r\n"),
			maxBody: maxBody, mustErr: true, wantErr: errInvalidRequest,
		},
		{
			id: "H1-02", name: "duplicate_cl_desync",
			payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\nContent-Length: 5\r\n\r\nhello"),
			maxBody: maxBody, mustErr: true, wantErr: errInvalidRequest,
		},
		{
			id: "H1-03", name: "cl_zero_smuggled_tail",
			payload: []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\n\r\nSMUGGLED"),
			maxBody: maxBody, mustOK: true,
		},
		{
			id: "H1-04", name: "obs_fold_header_injection",
			payload: []byte("POST /track HTTP/1.1\r\nX-Forwarded-For: 1.2.3.4\r\n injected: evil\r\nContent-Length: 0\r\n\r\n"),
			maxBody: maxBody, mustErr: true, wantErr: errInvalidRequest,
		},
		{
			id: "H1-05", name: "tls_hash_oversize_header",
			payload: append(append([]byte("POST /track HTTP/1.1\r\nX-TLS-Hash: "), bytes.Repeat([]byte("a"), 10240)...), []byte("\r\nContent-Length: 0\r\n\r\n")...),
			maxBody: maxBody, mustErr: true, wantErr: errInvalidRequest,
		},
		{
			id: "H1-06", name: "sec_ch_ua_ua_mismatch_still_parses",
			payload: []byte("POST /track HTTP/1.1\r\nUser-Agent: Chrome/120\r\nSec-CH-UA: \"Safari\";v=\"17\"\r\nContent-Length: 0\r\n\r\n"),
			maxBody: maxBody, mustOK: true,
		},
		{
			id: "H1-07", name: "click_spam_pipelined_50",
			payload: bytes.Repeat(minimalPOST, 50),
			maxBody: maxBody, mustOK: true,
		},
		{
			id: "H1-09", name: "x_original_method_ignored",
			payload: []byte("POST /track HTTP/1.1\r\nX-Original-Method: GET\r\nContent-Length: 0\r\n\r\n"),
			maxBody: maxBody, mustOK: true,
		},
		{
			id: "H1-10", name: "http_10_downgrade",
			payload: []byte("POST /track HTTP/1.0\r\nContent-Length: 0\r\n\r\n"),
			maxBody: maxBody, mustErr: true, wantErr: errInvalidRequest,
		},
		{
			id: "H1-11", name: "homoglyph_cyrillic_track_path",
			payload: []byte("POST /tr\u0430ck HTTP/1.1\r\nContent-Length: 0\r\n\r\n"),
			maxBody: maxBody, mustErr: true, wantErr: errInvalidRequest,
		},
		{
			id: "H1-12", name: "xff_long_chain",
			payload: []byte("POST /track HTTP/1.1\r\nX-Forwarded-For: ::1, 203.0.113.1, 10.0.0.1, 192.0.2.1\r\nContent-Length: 0\r\n\r\n"),
			maxBody: maxBody, mustOK: true,
		},
		{
			id: "H1-13", name: "te_chunked_with_cl_zero",
			payload: []byte("POST /track HTTP/1.1\r\nTransfer-Encoding: chunked\r\nContent-Length: 0\r\n\r\n"),
			maxBody: maxBody, mustErr: true, wantErr: errInvalidRequest,
		},
		{
			id: "H1-14", name: "cl_tab_prefix",
			payload: []byte("POST /track HTTP/1.1\r\nContent-Length:\t5\r\n\r\nhello"),
			maxBody: maxBody, mustOK: true,
		},
	}
}
