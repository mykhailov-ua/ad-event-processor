package ingestion

import (
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/ingestion/pb"
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/panjf2000/gnet/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChaos_TE_TE_Obfuscation(t *testing.T) {
	const maxBody = int64(1 << 20)
	vectors := []struct {
		name string
		wire []byte
	}{
		{
			name: "vt_before_chunked",
			wire: []byte("POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding:\x0bchunked\r\n\r\n0\r\n\r\n"),
		},
		{
			name: "duplicate_te_lines",
			wire: []byte("POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding: chunked\r\nTransfer-Encoding: gzip\r\n\r\n0\r\n\r\n"),
		},
		{
			name: "chunked_identity",
			wire: []byte("POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding: chunked, identity\r\n\r\n0\r\n\r\n"),
		},
		{
			name: "short_te_duplicate",
			wire: []byte("POST /openrtb/bid HTTP/1.1\r\nTE: chunked\r\nTE: gzip\r\n\r\n0\r\n\r\n"),
		},
	}
	for _, tc := range vectors {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			_, _, err := parseHTTP1(tc.wire, maxBody, nil)
			elapsed := time.Since(start)
			require.ErrorIs(t, err, errInvalidRequest)
			require.Less(t, elapsed, 50*time.Microsecond)
		})
	}
	faultproof.Log(t, "parser_security_ps_g05", map[string]string{
		"gap_id":  "PS-G05",
		"gap":     "closed",
		"vectors": "4",
	})
}

func TestChaos_Proto_FieldBudget(t *testing.T) {
	wire := chaosProtoWireFieldFlood(10_000)
	var evt pb.AdEvent
	start := time.Now()
	err := unmarshalAdEventVT(&evt, wire)
	elapsed := time.Since(start)
	require.ErrorIs(t, err, errProtoFieldBudget)
	require.Less(t, elapsed, 50*time.Microsecond)

	allocs := testing.AllocsPerRun(20, func() {
		var e pb.AdEvent
		_ = unmarshalAdEventVT(&e, wire)
	})
	assert.Equal(t, float64(0), allocs)

	faultproof.Log(t, "parser_security_ps_g06", map[string]string{
		"gap_id":     "PS-G06",
		"gap":        "closed",
		"fields":     "10000",
		"elapsed_ns": elapsed.String(),
	})
}

func TestChaos_HPACK_ContinuationBomb(t *testing.T) {
	block := make([]byte, 0, 65)
	block = append(block, 0xFF)
	for i := 0; i < 63; i++ {
		block = append(block, 0xFF)
	}
	block = append(block, 0x00)
	wire := buildH2WireAfterPreface(buildH2HeadersDataFrames(1, block, nil))
	cfg := &config.Config{MaxRequestBodySize: 1 << 20, H2IncompleteMax: 3}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	conn := NewGnetHarnessConn(wire)

	start := time.Now()
	act := h.OnTraffic(conn)
	elapsed := time.Since(start)
	require.Equal(t, gnet.Close, act)
	require.Less(t, elapsed, 50*time.Microsecond)

	faultproof.Log(t, "parser_security_ps_g07", map[string]string{
		"gap_id":     "PS-G07",
		"gap":        "closed",
		"elapsed_ns": elapsed.String(),
	})
}
