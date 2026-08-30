package ingest

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

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
	faultproof.Log(t, "parser_security", map[string]string{
		"case_id": "json_pool_cap_poisoning",
		"proof":   "closed",
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

func TestChaos_ParserSecurity_KeyPairFlood(t *testing.T) {
	body := buildJSONKeyPairSpam(MaxJSONKeyPairs + 1)
	start := time.Now()

	var req TrackRequest
	err := ParseTrackRequestJSON(&req, body)
	require.Error(t, err)
	require.Less(t, time.Since(start), 50*time.Millisecond)

	var parsed OpenRTB3Parsed
	ok := parseOpenRTB3FSMInto(&parsed, body)
	require.False(t, ok)

	faultproof.Log(t, "parser_security_json_key_pair_flood", map[string]string{
		"case_id": "json_key_pair_flood",
		"proof":   "closed",
	})
}

func TestChaos_ParserSecurity_KeyEscapeWalk(t *testing.T) {
	key := strings.Repeat(`\\`, MaxJSONStringScanBytes+1)
	body := []byte(`{"` + key + `":1}`)
	start := time.Now()

	var parsed OpenRTB3Parsed
	ok := parseOpenRTB3FSMInto(&parsed, body)
	require.False(t, ok)
	require.Less(t, time.Since(start), 2*time.Second)

	faultproof.Log(t, "parser_security_json_key_escape_walk", map[string]string{
		"case_id": "json_key_escape_walk",
		"proof":   "closed",
	})
}

func TestChaos_ParserSecurity_OverlongUTF8(t *testing.T) {
	jsonStrictUTF8Enabled.Store(true)
	t.Cleanup(func() { jsonStrictUTF8Enabled.Store(true) })

	body := []byte(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000","user_id":"` + "\xc0\xaf" + `"}`)
	var req TrackRequest
	err := ParseTrackRequestJSON(&req, body)
	require.Error(t, err)

	faultproof.Log(t, "parser_security_json_overlong_utf8", map[string]string{
		"case_id": "json_overlong_utf8",
		"proof":   "closed",
	})
}
