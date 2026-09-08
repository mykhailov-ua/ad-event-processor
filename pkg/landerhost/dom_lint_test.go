package landerhost

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomLint_holdoutRejectsMetaRefresh(t *testing.T) {
	html := []byte(`<!doctype html><html><head><meta http-equiv="refresh" content="0;url=https://evil.test"></head><body>ok</body></html>`)
	v := LintAsset("index.html", html)
	require.NotEmpty(t, v)
	assert.Equal(t, DomLintRuleMetaRefresh, v[0].Rule)
}

func TestDomLint_holdoutRejectsHiddenOverlay(t *testing.T) {
	css := []byte(`.trap{display:none;position:fixed;width:100%;height:100vh;top:0;left:0}`)
	v := LintAsset("overlay.css", css)
	require.NotEmpty(t, v)
	assert.Equal(t, DomLintRuleHiddenOverlay, v[0].Rule)
}

func TestDomLint_holdoutRejectsLocationChain(t *testing.T) {
	js := []byte(`window.location='https://a.test'; setTimeout(function(){location.href='https://b.test'},1);`)
	v := LintAsset("redirect.js", js)
	require.NotEmpty(t, v)
	assert.Equal(t, DomLintRuleLocationChain, v[0].Rule)
}

func TestDomLint_holdoutAllowsCleanLander(t *testing.T) {
	html := []byte(`<!doctype html><html><head><title>ok</title></head><body><a href="/offer">go</a></body></html>`)
	assert.Empty(t, LintAsset("index.html", html))
}
