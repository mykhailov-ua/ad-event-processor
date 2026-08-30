package ingest

import (
	"fmt"
	"strings"
	"testing"

	"ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/require"
)

func TestChaos_ParserSecurity_UnicodeKeyRejected(t *testing.T) {
	key := "campa" + string([]byte{0xC5, 0xBF}) + "gn_id"
	body := []byte(fmt.Sprintf(`{"%s":"550e8400-e29b-41d4-a716-446655440000"}`, key))
	var req TrackRequest
	err := ParseTrackRequestJSON(&req, body)
	require.Error(t, err, "non-ASCII key must be rejected")
	faultproof.Log(t, "parser_security_json_non_ascii_key", map[string]string{
		"case_id": "json_non_ascii_key",
		"proof":   "closed",
	})
}

func TestChaos_ParserSecurity_DuplicateKeyLastWins(t *testing.T) {
	body := []byte(`{"id":"first","id":"second","imp":[{"id":"imp-b"}]}`)
	parsed := ParseOpenRTB26(body)
	require.True(t, parsed.OK)
	require.Equal(t, "second", string(parsed.RequestID[:parsed.RequestIDLen]))
	require.Equal(t, "imp-b", string(parsed.ImpID[:parsed.ImpIDLen]))
	faultproof.Log(t, "parser_security_openrtb_duplicate_keys", map[string]string{
		"case_id": "openrtb_duplicate_keys",
		"proof":   "closed",
	})
}

func TestChaos_ParserSecurity_LoneSurrogateRejected(t *testing.T) {
	body := []byte(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000","user_id":"\uD800"}`)
	var req TrackRequest
	err := ParseTrackRequestJSON(&req, body)
	require.Error(t, err, "lone high surrogate must reject parse")
	faultproof.Log(t, "parser_security_json_lone_surrogate", map[string]string{
		"case_id": "json_lone_surrogate",
		"proof":   "closed",
	})
}

func TestChaos_ParserSecurity_DistributedWSBomb(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000"`)
	for range MaxJSONTotalWSkip + 64 {
		b.WriteByte(' ')
	}
	b.WriteString(`,"type":"click"}`)
	var req TrackRequest
	err := ParseTrackRequestJSON(&req, []byte(b.String()))
	require.Error(t, err, "distributed whitespace bomb must be rejected")
	faultproof.Log(t, "parser_security_json_whitespace_bomb", map[string]string{
		"case_id": "json_whitespace_bomb",
		"proof":   "closed",
	})
}

func TestChaos_ParserSecurity_QuoteDenseStringRejected(t *testing.T) {
	escapes := strings.Repeat(`\"`, MaxJSONStringEscapes+1)
	body := fmt.Sprintf(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000","user_id":"%s"}`, escapes)
	var req TrackRequest
	err := ParseTrackRequestJSON(&req, []byte(body))
	require.Error(t, err)
	faultproof.Log(t, "parser_security_json_quote_escape_bomb", map[string]string{
		"case_id": "json_quote_escape_bomb",
		"proof":   "closed",
	})
}

func TestChaos_ParserSecurity_NestedPayloadEscapeBomb(t *testing.T) {
	escapes := strings.Repeat(`\"`, MaxJSONStringEscapes+1)
	body := fmt.Sprintf(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000","payload":{"x":"%s"}}`, escapes)
	var req TrackRequest
	err := ParseTrackRequestJSON(&req, []byte(body))
	require.Error(t, err, "nested payload escape bomb must honor MaxJSONStringEscapes")
	faultproof.Log(t, "parser_security_json_quote_escape_bomb", map[string]string{
		"case_id": "json_quote_escape_bomb",
		"proof":   "closed",
		"path":    "payload",
	})
}

func TestChaos_ParserSecurity_ValueLiteralBomb(t *testing.T) {
	digits := strings.Repeat("1", MaxJSONStringScanBytes+1)
	body := []byte(fmt.Sprintf(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000","noise":%s}`, digits))
	var req TrackRequest
	err := ParseTrackRequestJSON(&req, body)
	require.Error(t, err, "oversized numeric literal in skipped value must be rejected")
	faultproof.Log(t, "parser_security_json_value_literal_bomb", map[string]string{
		"case_id": "json_value_literal_bomb",
		"proof":   "closed",
	})
}

func TestSkipJSONValueBudget_nestedRawControlRejected(t *testing.T) {
	body := []byte("{\"payload\":{\"a\":\"" + "\n" + "\"}}")
	bud := newJSONScanBudget()
	_, err := skipJSONValueBudget(body, 0, &bud)
	require.Error(t, err)
}
