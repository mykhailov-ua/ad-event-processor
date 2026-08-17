package ingestion

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestParseTgQuery(t *testing.T) {
	t.Parallel()
	cid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	clickID := uuid.MustParse("d5671191-236b-4e94-825e-399185a9bc8d")

	t.Run("Valid parameters", func(t *testing.T) {
		path := []byte("/tg/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&click_id=d5671191-236b-4e94-825e-399185a9bc8d&bridge_token=token_abc123_&premium=1&motivated=true&sub1=testsub")
		parsed := &tgQueryParsed{}
		_ = parseTgQuery(path, nil, parsed)
		require.True(t, parsed.ok)
		require.Equal(t, cid, parsed.campaignID)
		require.Equal(t, clickID, parsed.clickID)
		require.Equal(t, "token_abc123_", parsed.bridgeToken)
		require.True(t, parsed.premium)
		require.True(t, parsed.motivated)
		require.Equal(t, "testsub", parsed.subs[0])
	})

	t.Run("dmr query flag", func(t *testing.T) {
		path := []byte("/tg/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&click_id=d5671191-236b-4e94-825e-399185a9bc8d&bridge_token=token_abc123_&dmr=1")
		parsed := &tgQueryParsed{}
		_ = parseTgQuery(path, nil, parsed)
		require.True(t, parsed.ok)
		require.True(t, parsed.dmr)
	})

	t.Run("Forbidden parameters immediately reject", func(t *testing.T) {
		forbiddenPaths := [][]byte{
			[]byte("/tg/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&click_id=d5671191-236b-4e94-825e-399185a9bc8d&hash=something"),
			[]byte("/tg/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&click_id=d5671191-236b-4e94-825e-399185a9bc8d&user=something"),
			[]byte("/tg/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&click_id=d5671191-236b-4e94-825e-399185a9bc8d&initData=something"),
			[]byte("/tg/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&click_id=d5671191-236b-4e94-825e-399185a9bc8d&auth_date=something"),
			[]byte("/tg/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&click_id=d5671191-236b-4e94-825e-399185a9bc8d&signature=something"),
		}
		for _, p := range forbiddenPaths {
			parsed := &tgQueryParsed{}
			_ = parseTgQuery(p, nil, parsed)
			require.False(t, parsed.ok, "should reject path: %s", string(p))
		}
	})
}

func TestBuildTgRedirectLocation(t *testing.T) {
	t.Parallel()
	base := []byte("https://my-app.com/start?click_id={click_id}&token={bridge_token}&sub={sub1}")
	loc, ok := buildTgRedirectLocation(nil, base, "d5671191-236b-4e94-825e-399185a9bc8d", "bridge_abc123", [5]string{"subval"}, []byte("extra=1"))
	require.True(t, ok)
	require.Equal(t, "https://my-app.com/start?click_id=d5671191-236b-4e94-825e-399185a9bc8d&token=bridge_abc123&sub=subval&extra=1", string(loc))
}

func TestBuildTgRedirectLocation_encodesMacroValues(t *testing.T) {
	t.Parallel()
	base := []byte("https://my-app.com/start?sub={sub1}&token={bridge_token}")
	loc, ok := buildTgRedirectLocation(nil, base, "click-id", "a&b=c", [5]string{"x y"}, nil)
	require.True(t, ok)
	require.Equal(t, "https://my-app.com/start?sub=x%20y&token=a%26b%3Dc", string(loc))
}

func FuzzParseTgClickQuery(f *testing.F) {
	f.Add([]byte("/tg/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&click_id=d5671191-236b-4e94-825e-399185a9bc8d&bridge_token=token_abc123_"))
	f.Fuzz(func(t *testing.T, path []byte) {
		parsed := &tgQueryParsed{}
		scratch := make([]byte, 0, 512)
		_ = parseTgQuery(path, scratch, parsed)
		if parsed.ok {
			if parsed.campaignID == uuid.Nil || parsed.clickID == uuid.Nil {
				t.Fatalf("ok without required ids: %q", path)
			}
			if parsed.bridgeToken != "" && !validateBridgeToken([]byte(parsed.bridgeToken)) {
				t.Fatalf("ok with invalid bridge_token: %q", parsed.bridgeToken)
			}
		}
	})
}

func BenchmarkParseTgQuery_ZeroAlloc(b *testing.B) {
	path := []byte("/tg/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&click_id=d5671191-236b-4e94-825e-399185a9bc8d&bridge_token=token_abc123_&premium=1&motivated=true&sub1=testsub")
	parsed := &tgQueryParsed{}
	scratch := make([]byte, 0, 512)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scratch = parseTgQuery(path, scratch[:0], parsed)
	}
}
