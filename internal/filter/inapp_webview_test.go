package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebView_holdoutInstagramClassification(t *testing.T) {
	platform, class := InAppWebViewClassification("Instagram 300.0.0.0 Android")
	assert.Equal(t, "instagram", platform)
	assert.Equal(t, InAppWebViewClassInstagram, class)
}

func TestWebView_holdoutDesktopChromeNotClassified(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	platform, class := InAppWebViewClassification(ua)
	assert.Equal(t, InAppWebViewClassNone, class)
	assert.Empty(t, platform)
}
