package campaign

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaybeSignProgrammaticClickURL_holdoutSkipsWithoutSecret(t *testing.T) {
	t.Parallel()
	base := "https://trk.example.com/click?campaign_id=11111111-1111-7111-8111-111111111111&click_id=clk-1"
	got := MaybeSignProgrammaticClickURL(base, "clk-1", ProgrammaticClickLinkSigning{
		Enabled: true,
		TTLSec:  300,
	})
	assert.Equal(t, base, got)
}

func TestMaybeSignProgrammaticClickURL_appendsExpiresAndSig(t *testing.T) {
	t.Parallel()
	base := "https://trk.example.com/click?campaign_id=11111111-1111-7111-8111-111111111111&click_id=clk-1"
	secret := []byte("test-link-signing-secret")
	got := MaybeSignProgrammaticClickURL(base, "clk-1", ProgrammaticClickLinkSigning{
		Enabled: true,
		TTLSec:  300,
		Secret:  secret,
	})
	require.Contains(t, got, "expires=")
	require.Contains(t, got, "_sig=")
	assert.True(t, strings.HasPrefix(got, base+"&"))
}
