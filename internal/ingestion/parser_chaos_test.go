package ingestion

import (
	"fmt"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/openrtb"
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chaosChunkedHTTP1(body []byte, chunkCount int) []byte {
	if chunkCount < 1 {
		chunkCount = 1
	}
	var wire []byte
	wire = append(wire, "POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding: chunked\r\nContent-Type: application/json\r\n\r\n"...)
	if len(body) == 0 {
		return append(wire, "0\r\n\r\n"...)
	}
	chunkSize := len(body) / chunkCount
	if chunkSize == 0 {
		chunkSize = 1
	}
	emitted := 0
	remaining := chunkCount
	for emitted < len(body) && remaining > 0 {
		end := emitted + chunkSize
		if end > len(body) {
			end = len(body)
		}
		piece := body[emitted:end]
		wire = append(wire, fmt.Sprintf("%x\r\n", len(piece))...)
		wire = append(wire, piece...)
		wire = append(wire, "\r\n"...)
		emitted = end
		remaining--
	}
	if emitted < len(body) {
		piece := body[emitted:]
		wire = append(wire, fmt.Sprintf("%x\r\n", len(piece))...)
		wire = append(wire, piece...)
		wire = append(wire, "\r\n"...)
	}
	return append(wire, "0\r\n\r\n"...)
}

func TestChaos_ParserIngress_2026(t *testing.T) {
	t.Run("C-H04_chunked_50_fragments", func(t *testing.T) {
		body := []byte(`{"id":"req-1","imp":[{"id":"1"}]}`)
		wire := chaosChunkedHTTP1(body, 50)
		const maxBody = int64(1 << 20)
		var scratch []byte
		start := time.Now()
		n, req, err := parseHTTP1(wire, maxBody, &scratch)
		elapsed := time.Since(start)
		require.NoError(t, err)
		require.Equal(t, len(wire), n)
		require.Equal(t, body, req.Body)
		allocs := testing.AllocsPerRun(20, func() {
			_, _, err := parseHTTP1(wire, maxBody, &scratch)
			if err != nil {
				t.Fatal(err)
			}
		})
		assert.Equal(t, float64(0), allocs)
		faultproof.Log(t, "parser_chaos_c_h04_chunked", map[string]string{
			"fragments":  "50",
			"elapsed_ns": elapsed.String(),
			"allocs":     fmt.Sprintf("%.0f", allocs),
		})
	})

	t.Run("C-O04_ortb26_garbage_64k", func(t *testing.T) {
		garbage := make([]byte, 64*1024)
		for i := range garbage {
			garbage[i] = 'A'
		}
		var hot OpenRTB26Hot
		var cold OpenRTB26Cold
		start := time.Now()
		ParseOpenRTB26Split(garbage, &hot, &cold)
		elapsed := time.Since(start)
		require.False(t, hot.ExchangeReady(openrtb.ExchangeConfig{}))
		assert.Less(t, elapsed, 50*time.Microsecond, "64k garbage parse took %v", elapsed)
		faultproof.Log(t, "parser_chaos_c_o04_garbage", map[string]string{
			"bytes":      "65536",
			"elapsed_ns": elapsed.String(),
		})
	})

	t.Run("C-N02_JSON_whitespace_prefix", func(t *testing.T) {
		data := chaosWSBomb(1<<20, chaosValidTrackJSON)
		var tr TrackRequest
		start := time.Now()
		err := ParseTrackRequestJSON(&tr, data)
		elapsed := time.Since(start)
		require.ErrorIs(t, err, ErrMalformed)
		assert.Less(t, elapsed, time.Millisecond, "WS bomb reject took %v", elapsed)
		faultproof.Log(t, "parser_chaos_c_n02_json", map[string]string{
			"prefix_bytes": "1048576",
			"elapsed_ns":   elapsed.String(),
			"allocs":       "0",
		})
	})

	t.Run("C-N02_JSON_whitespace_only", func(t *testing.T) {
		data := chaosWSBomb(1<<20, "")
		var tr TrackRequest
		start := time.Now()
		err := ParseTrackRequestJSON(&tr, data)
		elapsed := time.Since(start)
		require.ErrorIs(t, err, ErrMalformed)
		assert.Less(t, elapsed, time.Millisecond)
		faultproof.Log(t, "parser_chaos_c_n02_json_only", map[string]string{
			"bytes":      "1048576",
			"elapsed_ns": elapsed.String(),
		})
	})

	t.Run("C-N02_JSON_whitespace_under_cap", func(t *testing.T) {
		data := chaosWSBomb(MaxWSkip, chaosValidTrackJSON)
		var tr TrackRequest
		require.NoError(t, ParseTrackRequestJSON(&tr, data))
	})

	t.Run("C-N02_OpenRTB3_whitespace_prefix", func(t *testing.T) {
		suffix := `{"openrtb":{"request":{"item":[{"id":"550e8400-e29b-41d4-a716-446655440000"}]}}}`
		data := chaosWSBomb(1<<20, suffix)
		start := time.Now()
		ok := parseOpenRTB3FSMInto(&OpenRTB3Parsed{}, data)
		elapsed := time.Since(start)
		require.False(t, ok)
		assert.Less(t, elapsed, time.Millisecond)
		faultproof.Log(t, "parser_chaos_c_n02_ortb3", map[string]string{
			"prefix_bytes": "1048576",
			"elapsed_ns":   elapsed.String(),
		})
	})
}

func chaosQuoteDenseORTB(quotes int) []byte {
	payload := make([]byte, 0, quotes+64)
	for range quotes {
		payload = append(payload, '"')
	}
	payload = append(payload, `,"imp":[{"id":"1"}],"id":"req"}`...)
	return payload
}

func TestChaos_ORTB_QuoteDense1MB(t *testing.T) {
	payload := chaosQuoteDenseORTB(1 << 20)
	var hot OpenRTB26Hot
	var cold OpenRTB26Cold
	start := time.Now()
	ParseOpenRTB26Split(payload, &hot, &cold)
	elapsed := time.Since(start)
	require.False(t, hot.OK)
	require.Less(t, elapsed, 100*time.Microsecond)
	faultproof.Log(t, "parser_chaos_c_s04_quote_dense", map[string]string{
		"quotes":     "1048576",
		"elapsed_ns": elapsed.String(),
		"ok":         "false",
	})
}
