package ingestion

import (
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/pkg/faultproof"

	"github.com/panjf2000/gnet/v2"
	"github.com/stretchr/testify/require"
)

func TestChaos_ParserSecurity_PS_G01_SlowBodyStall(t *testing.T) {
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
	for i := 0; i < len(body)+2; i++ {
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

	gap := "open"
	if closed {
		gap = "closed"
	}
	faultproof.Log(t, "parser_security_ps_g01", map[string]string{
		"gap_id":      "http1_incomplete_body_spin_close",
		"gap":         gap,
		"conn_closed": boolStr(closed),
		"reason":      reason,
		"incomplete":  "true",
	})
	require.Equal(t, "closed", gap)
}

func TestChaos_ParserSecurity_PS_G02_ChunkExtCRLF(t *testing.T) {
	wire := []byte("POST /track HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n" +
		"5;foo\r\n" +
		"5\r\n" +
		"hello\r\n" +
		"0\r\n\r\n")
	_, _, err := parseHTTP1(wire, 1<<20, nil)
	gap := "open"
	if err != nil {
		gap = "closed"
	}
	faultproof.Log(t, "parser_security_ps_g02", map[string]string{
		"gap_id": "http1_chunked_te_smuggling",
		"gap":    gap,
		"err":    errString(err),
	})
	require.Equal(t, "closed", gap)
	require.ErrorIs(t, err, errInvalidRequest)
}

func TestChaos_ParserSecurity_PS_G03_QuoteDenseORTB(t *testing.T) {
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

	gap := "open"
	if !hot.OK && elapsed < 100*time.Microsecond {
		gap = "closed"
	}
	faultproof.Log(t, "parser_security_ps_g03", map[string]string{
		"gap_id":     "openrtb_scan_budget",
		"gap":        gap,
		"elapsed_ns": elapsed.String(),
		"ok":         boolStr(hot.OK),
	})
	require.Equal(t, "closed", gap)
	require.False(t, hot.OK)
}

func TestChaos_ParserSecurity_PS_G05_TETabObfuscation(t *testing.T) {
	wire := []byte("POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding:\x0bchunked\r\n\r\n0\r\n\r\n")
	_, _, err := parseHTTP1(wire, 1<<20, nil)
	require.ErrorIs(t, err, errInvalidRequest)
	faultproof.Log(t, "parser_security_ps_g05", map[string]string{
		"gap_id": "wire_parser_budget",
		"gap":    "closed",
		"err":    errString(err),
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
