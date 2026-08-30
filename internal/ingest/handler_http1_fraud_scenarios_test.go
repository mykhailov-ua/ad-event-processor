package ingest

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fraudHTTP1PostChecks = map[string]func(t *testing.T, n int, req parsedHTTPRequest, rest []byte){
	"H1-03": func(t *testing.T, n int, req parsedHTTPRequest, rest []byte) {
		assert.Equal(t, 0, req.ContentLength)
		assert.Equal(t, n, len("POST /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n"))
		assert.Equal(t, "SMUGGLED", string(rest))
	},
	"H1-07": func(t *testing.T, n int, req parsedHTTPRequest, rest []byte) {
		minimalPOST := []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n")
		assert.Equal(t, len(minimalPOST), n)
		assert.Equal(t, 49*len(minimalPOST), len(rest))
	},
	"H1-09": func(t *testing.T, n int, req parsedHTTPRequest, rest []byte) {
		assert.Equal(t, "POST", string(req.Method))
	},
	"H1-12": func(t *testing.T, n int, req parsedHTTPRequest, rest []byte) {
		assert.Contains(t, string(req.ClientIP), "203.0.113.1")
	},
	"H1-14": func(t *testing.T, n int, req parsedHTTPRequest, rest []byte) {
		assert.Equal(t, 5, req.ContentLength)
		assert.Equal(t, "hello", string(req.Body))
	},
}

func TestFraudScenarios_HTTP1_2026(t *testing.T) {
	var gaps []string
	for _, tc := range fraudHTTP1Cases2026() {
		tc := tc
		t.Run(tc.ID+"_"+tc.Name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					gaps = append(gaps, fmt.Sprintf("%s: PANIC %v", tc.ID, r))
					t.Fatalf("panic: %v", r)
				}
			}()

			n, req, err := parseHTTP1(tc.Payload, tc.MaxBody, nil)
			rest := tc.Payload[n:]

			switch {
			case tc.MustErr:
				if err == nil {
					msg := fmt.Sprintf("%s [%s]: holdout expected reject (%v) got success n=%d rest=%q", tc.ID, tc.Name, tc.WantErr, n, truncateBytes(rest, 40))
					gaps = append(gaps, msg)
					t.Fatal(msg)
				}
				if tc.WantErr != nil && !assert.ErrorIs(t, err, tc.WantErr) {
					msg := fmt.Sprintf("%s [%s]: holdout expected %v got %v", tc.ID, tc.Name, tc.WantErr, err)
					gaps = append(gaps, msg)
				}
			case tc.WantOK:
				if err != nil {
					msg := fmt.Sprintf("%s [%s]: holdout expected accept got %v", tc.ID, tc.Name, err)
					gaps = append(gaps, msg)
					t.Fatal(msg)
				}
				require.Greater(t, n, 0)
				if postCheck := fraudHTTP1PostChecks[tc.ID]; postCheck != nil {
					postCheck(t, n, req, rest)
				}
			}
		})
	}
	if len(gaps) > 0 {
		t.Logf("fraud_http1_gaps=%d", len(gaps))
		for _, g := range gaps {
			t.Log(g)
		}
	}
	faultproof.Log(t, "fraud_http1_2026", map[string]string{
		"cases": fmt.Sprintf("%d", len(fraudHTTP1Cases2026())),
		"gaps":  fmt.Sprintf("%d", len(gaps)),
	})
}

func TestFraudScenarios_HTTP1_PipelineSpam(t *testing.T) {
	const maxBody = int64(1024)
	reqLine := []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n")
	buf := bytes.Repeat(reqLine, 50)
	offset := 0
	for i := range 50 {
		n, _, err := parseHTTP1(buf[offset:], maxBody, nil)
		require.NoError(t, err, "pipeline iter %d", i)
		require.Equal(t, len(reqLine), n)
		offset += n
	}
	require.Equal(t, len(buf), offset)
}

func TestFraudScenarios_HTTP1_HeaderValueCRLFInjection(t *testing.T) {
	payload := []byte("POST /track HTTP/1.1\r\nX-Evil: safe\r\n continuation\r\nContent-Length: 0\r\n\r\n")
	_, _, err := parseHTTP1(payload, 1024, nil)
	if err == nil {
		t.Fatal("obs-fold continuation in header value accepted")
	}
}

func TestFraudScenarios_HTTP1_FraudHeaderBounds(t *testing.T) {
	const maxHeader = 1024
	h := strings.Repeat("x", maxHeader+1)
	payload := fmt.Appendf(nil, "POST /track HTTP/1.1\r\nX-TLS-Hash: %s\r\nContent-Length: 0\r\n\r\n", h)
	_, _, err := parseHTTP1(payload, 1024, nil)
	require.ErrorIs(t, err, errInvalidRequest)
}
