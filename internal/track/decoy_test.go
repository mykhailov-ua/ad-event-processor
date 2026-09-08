package track

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestParseHostedLanderIDFromURL(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	got := ParseHostedLanderIDFromURL("https://trk.example.com/lp/" + id.String() + "/index.html")
	require.Equal(t, id, got)
	require.Equal(t, uuid.Nil, ParseHostedLanderIDFromURL("https://safe.example/white"))
}

func TestDecoyTemplate_holdout_hostedSHA256NotStatic(t *testing.T) {
	landerID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	static := BuildDecoyBodyNoMetric(DecoyTemplateInput{})
	hosted := BuildDecoyBodyNoMetric(DecoyTemplateInput{DecoyLanderID: landerID})
	require.NotEqual(t, sha256Hex(static), sha256Hex(hosted))
	require.Contains(t, string(hosted), "/lp/"+landerID.String()+"/")
	require.NotContains(t, string(static), "<iframe")
}

func TestDecoyTemplate_derivesHostedFromSafePageURL(t *testing.T) {
	landerID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	source, url := ResolveDecoyTemplate(DecoyTemplateInput{
		SafePageURL: "https://trk.example.com/lp/" + landerID.String() + "/",
	})
	require.Equal(t, DecoySourceHosted, source)
	require.Equal(t, []byte(HostedLanderPath(landerID)), url)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
