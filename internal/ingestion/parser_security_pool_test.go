package ingestion

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/require"
)

func TestRequestBufferPool_NoCapPoisoning(t *testing.T) {
	large := make([]byte, maxPoolObjectSize+1)
	largePtr := &large
	putRequestBuffer(largePtr)

	small := make([]byte, 512)
	smallPtr := &small
	putRequestBuffer(smallPtr)

	got := requestBufferPool.Get().(*[]byte)
	gotCap := cap(*got)
	putRequestBuffer(got)

	require.LessOrEqual(t, gotCap, maxPoolObjectSize)
	faultproof.Log(t, "parser_security_ps_h01", map[string]string{
		"gap_id":  "PS-H01",
		"gap":     "closed",
		"max_cap": strconv.Itoa(gotCap),
	})
}

func buildJSONKeyPairSpam(pairs int) []byte {
	var b strings.Builder
	b.Grow(pairs * 8)
	b.WriteByte('{')
	for i := range pairs {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`"a":0`)
	}
	b.WriteByte('}')
	return []byte(b.String())
}

func TestChaos_ParserSecurity_PS_H02_KeyPairFlood(t *testing.T) {
	body := buildJSONKeyPairSpam(MaxJSONKeyPairs + 1)
	start := time.Now()

	var req TrackRequest
	err := ParseTrackRequestJSON(&req, body)
	require.Error(t, err)
	require.Less(t, time.Since(start), 50*time.Millisecond)

	var parsed OpenRTB3Parsed
	ok := parseOpenRTB3FSMInto(&parsed, body)
	require.False(t, ok)

	faultproof.Log(t, "parser_security_ps_h02", map[string]string{
		"gap_id": "PS-H02",
		"gap":    "closed",
	})
}

func TestChaos_ParserSecurity_PS_H04_KeyEscapeWalk(t *testing.T) {
	key := strings.Repeat(`\\`, MaxJSONStringScanBytes+1)
	body := []byte(`{"` + key + `":1}`)
	start := time.Now()

	var parsed OpenRTB3Parsed
	ok := parseOpenRTB3FSMInto(&parsed, body)
	require.False(t, ok)
	require.Less(t, time.Since(start), 2*time.Second)

	faultproof.Log(t, "parser_security_ps_h04", map[string]string{
		"gap_id": "PS-H04",
		"gap":    "closed",
	})
}

func TestChaos_ParserSecurity_PS_H05_OverlongUTF8(t *testing.T) {
	jsonStrictUTF8Enabled.Store(true)
	t.Cleanup(func() { jsonStrictUTF8Enabled.Store(true) })

	body := []byte(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000","user_id":"` + "\xc0\xaf" + `"}`)
	var req TrackRequest
	err := ParseTrackRequestJSON(&req, body)
	require.Error(t, err)

	faultproof.Log(t, "parser_security_ps_h05", map[string]string{
		"gap_id": "PS-H05",
		"gap":    "closed",
	})
}
