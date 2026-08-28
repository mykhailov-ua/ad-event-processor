package campaign

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpand_redirectMacros_holdout(t *testing.T) {
	t.Parallel()
	ctx := PreviewContext("camp-1", PreviewRequest{Sub1: "alpha"})
	out, unresolved := Expand("https://offer.example/lp?cid={click_id}&src={sub1}&uid={user_id}", ctx)
	assert.Empty(t, unresolved)
	assert.Contains(t, out, previewClickID)
	assert.Contains(t, out, "alpha")
	assert.Contains(t, out, previewUserID)
}

func TestExpand_presetDoubleBrace_holdout(t *testing.T) {
	t.Parallel()
	ctx := PreviewContext("camp-1", PreviewRequest{Sub1: "alpha", Country: "US"})
	out, unresolved := Expand("https://offer.example/click?cid={{campaign.id}}&sub1={{sub1}}&cc={{country}}", ctx)
	assert.Empty(t, unresolved)
	assert.Contains(t, out, "camp-1")
	assert.Contains(t, out, "alpha")
	assert.Contains(t, out, "US")
}

func TestExpand_unresolvedDoubleBrace_holdout(t *testing.T) {
	t.Parallel()
	ctx := PreviewContext("camp-1", PreviewRequest{})
	out, unresolved := Expand("https://offer.example/?x={{adset.id}}&y={{sub1}}", ctx)
	require.Len(t, unresolved, 2)
	assert.Contains(t, out, "{{adset.id}}")
	assert.Contains(t, out, "{{sub1}}")
}

func TestExpand_unclosedBraceLeavesLiteral(t *testing.T) {
	t.Parallel()
	ctx := PreviewContext("camp-1", PreviewRequest{})
	out, unresolved := Expand("https://offer.example/{click_id", ctx)
	assert.Empty(t, unresolved)
	assert.Equal(t, "https://offer.example/{click_id", out)
}
