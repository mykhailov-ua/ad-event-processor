package ingest

import (
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/pkg/faultproof"

	"github.com/panjf2000/gnet/v2"
	"github.com/stretchr/testify/require"
)

func TestChaos_ParserSecurity_SlowBodyStall(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize: 1 << 20,
		HTTP1IncompleteMax: 3,
		HTTP1BodyIdleMs:    60_000,
	}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	conn := newFaultGnetConn()
	conn.Append(chaosSlowBodyHeaders())
	require.Equal(t, gnet.None, h.OnTraffic(conn))

	closed := false
	reason := ""
	body := chaosSlowBodyPrefixBytes()
	for i := range len(body) + 2 {
		if i < len(body) {
			conn.Append(body[i : i+1])
		}
		act := h.OnTraffic(conn)
		if act == gnet.Close {
			closed = true
			reason = "spin"
			break
		}
	}

	proof := "open"
	if closed {
		proof = "closed"
	}
	faultproof.Log(t, "parser_security_http1_incomplete_body_spin_close", map[string]string{
		"case_id":     "http1_incomplete_body_spin_close",
		"proof":       proof,
		"conn_closed": boolStr(closed),
		"reason":      reason,
		"incomplete":  "true",
	})
	require.Equal(t, "closed", proof)
}

func TestChaos_ParserSecurity_ChunkExtCRLF(t *testing.T) {
	wire := []byte("POST /track HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n" +
		"5;foo\r\n" +
		"5\r\n" +
		"hello\r\n" +
		"0\r\n\r\n")
	_, _, err := parseHTTP1(wire, 1<<20, nil)
	proof := "open"
	if err != nil {
		proof = "closed"
	}
	faultproof.Log(t, "parser_security_http1_chunked_te_smuggling", map[string]string{
		"case_id": "http1_chunked_te_smuggling",
		"proof":   proof,
		"err":     errString(err),
	})
	require.Equal(t, "closed", proof)
	require.ErrorIs(t, err, errInvalidRequest)
}

func TestChaos_ParserSecurity_QuoteDenseORTB(t *testing.T) {
	const quotes = 1 << 20
	payload := make([]byte, 0, quotes+64)
	for range quotes {
		payload = append(payload, '"')
	}
	payload = append(payload, `,"imp":[{"id":"1"}],"id":"req"}`...)

	var hot OpenRTB26Hot
	var cold OpenRTB26Cold
	start := time.Now()
	ParseOpenRTB26Split(payload, &hot, &cold)
	elapsed := time.Since(start)

	proof := "open"
	if !hot.OK && elapsed < 100*time.Microsecond {
		proof = "closed"
	}
	faultproof.Log(t, "parser_security_openrtb_scan_budget", map[string]string{
		"case_id":    "openrtb_scan_budget",
		"proof":      proof,
		"elapsed_ns": elapsed.String(),
		"ok":         boolStr(hot.OK),
	})
	require.Equal(t, "closed", proof)
	require.False(t, hot.OK)
}

func TestChaos_ParserSecurity_TETabObfuscation(t *testing.T) {
	wire := []byte("POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding:\x0bchunked\r\n\r\n0\r\n\r\n")
	_, _, err := parseHTTP1(wire, 1<<20, nil)
	require.ErrorIs(t, err, errInvalidRequest)
	faultproof.Log(t, "parser_security_wire_parser_budget", map[string]string{
		"case_id": "wire_parser_budget",
		"proof":   "closed",
		"err":     errString(err),
	})
}

func boolStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
