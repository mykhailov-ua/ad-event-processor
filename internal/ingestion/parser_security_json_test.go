package ingestion

import (
	"fmt"
	"strings"
	"testing"

	"ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/require"
)

func TestChaos_ParserSecurity_PS_G09_UnicodeKeyRejected(t *testing.T) {
	key := "campa" + string([]byte{0xC5, 0xBF}) + "gn_id"
	body := []byte(fmt.Sprintf(`{"%s":"550e8400-e29b-41d4-a716-446655440000"}`, key))
	var req TrackRequest
	err := ParseTrackRequestJSON(&req, body)
	require.Error(t, err, "non-ASCII key must be rejected")
	faultproof.Log(t, "parser_security_ps_g09", map[string]string{
		"gap_id": "json_non_ascii_key",
		"gap":    "closed",
	})
}

func TestChaos_ParserSecurity_PS_G10_DuplicateKeyLastWins(t *testing.T) {
	body := []byte(`{"id":"first","id":"second","imp":[{"id":"imp-b"}]}`)
	parsed := ParseOpenRTB26(body)
	require.True(t, parsed.OK)
	require.Equal(t, "second", string(parsed.RequestID[:parsed.RequestIDLen]))
	require.Equal(t, "imp-b", string(parsed.ImpID[:parsed.ImpIDLen]))
	faultproof.Log(t, "parser_security_ps_g10", map[string]string{
		"gap_id": "openrtb_duplicate_keys",
		"gap":    "closed",
	})
}

func TestChaos_ParserSecurity_PS_G11_LoneSurrogateRejected(t *testing.T) {
	body := []byte(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000","user_id":"\uD800"}`)
	var req TrackRequest
	err := ParseTrackRequestJSON(&req, body)
	require.Error(t, err, "lone high surrogate must reject parse")
	faultproof.Log(t, "parser_security_ps_g11", map[string]string{
		"gap_id": "json_lone_surrogate",
		"gap":    "closed",
	})
}

func TestChaos_ParserSecurity_PS_G12_DistributedWSBomb(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000"`)
	for range MaxJSONTotalWSkip + 64 {
		b.WriteByte(' ')
	}
	b.WriteString(`,"type":"click"}`)
	var req TrackRequest
	err := ParseTrackRequestJSON(&req, []byte(b.String()))
	require.Error(t, err, "distributed whitespace bomb must be rejected")
	faultproof.Log(t, "parser_security_ps_g12", map[string]string{
		"gap_id": "json_whitespace_bomb",
		"gap":    "closed",
	})
}

func TestChaos_ParserSecurity_PS_G13_QuoteDenseStringRejected(t *testing.T) {
	escapes := strings.Repeat(`\"`, MaxJSONStringEscapes+1)
	body := fmt.Sprintf(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000","user_id":"%s"}`, escapes)
	var req TrackRequest
	err := ParseTrackRequestJSON(&req, []byte(body))
	require.Error(t, err)
	faultproof.Log(t, "parser_security_ps_g13", map[string]string{
		"gap_id": "json_quote_escape_bomb",
		"gap":    "closed",
	})
}

func TestChaos_ParserSecurity_PS_G13_NestedPayloadEscapeBomb(t *testing.T) {
	escapes := strings.Repeat(`\"`, MaxJSONStringEscapes+1)
	body := fmt.Sprintf(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000","payload":{"x":"%s"}}`, escapes)
	var req TrackRequest
	err := ParseTrackRequestJSON(&req, []byte(body))
	require.Error(t, err, "nested payload escape bomb must honor MaxJSONStringEscapes")
	faultproof.Log(t, "parser_security_ps_g13_nested", map[string]string{
		"gap_id": "json_quote_escape_bomb",
		"gap":    "closed",
		"path":   "payload",
	})
}

func TestSkipJSONValueBudget_nestedRawControlRejected(t *testing.T) {
	body := []byte("{\"payload\":{\"a\":\"" + "\n" + "\"}}")
	bud := newJSONScanBudget()
	_, err := skipJSONValueBudget(body, 0, &bud)
	require.Error(t, err)
}
