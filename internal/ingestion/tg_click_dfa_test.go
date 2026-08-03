package ingestion

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTgClick_RejectsInitDataParam(t *testing.T) {
	t.Parallel()
	parsed := &tgQueryParsed{}
	path := []byte("/tg/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&click_id=d5671191-236b-4e94-825e-399185a9bc8d&initData=bad")
	_ = parseTgQuery(path, nil, parsed)
	require.False(t, parsed.ok)
	t.Log("fault_proof fault=tg_click_initdata_reject")
}

func TestTgClickDFA_rejectsOversizedBridgeToken(t *testing.T) {
	t.Parallel()
	parsed := &tgQueryParsed{}
	token := make([]byte, 65)
	for i := range token {
		token[i] = 'a'
	}
	path := append([]byte("/tg/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&click_id=d5671191-236b-4e94-825e-399185a9bc8d&bridge_token="), token...)
	_ = parseTgQuery(path, nil, parsed)
	require.False(t, parsed.ok)
}

func TestTgClickDFA_rejectsInvalidBridgeCharset(t *testing.T) {
	t.Parallel()
	parsed := &tgQueryParsed{}
	path := []byte("/tg/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&click_id=d5671191-236b-4e94-825e-399185a9bc8d&bridge_token=bad%20token")
	_ = parseTgQuery(path, nil, parsed)
	require.False(t, parsed.ok)
}
