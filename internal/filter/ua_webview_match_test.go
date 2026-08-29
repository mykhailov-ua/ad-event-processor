package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUaMatchesInAppWebView(t *testing.T) {
	assert.True(t, uaMatchesInAppWebView("Mozilla/5.0 [FBAN/FB4A;FBAV/128.0.0.0;]"))
	assert.True(t, uaMatchesInAppWebView("musical_ly_12.0 Android"))
	assert.True(t, uaMatchesInAppWebView("Instagram 300.0.0.0"))
	assert.False(t, uaMatchesInAppWebView("Mozilla/5.0 Chrome/120"))
	assert.False(t, uaMatchesInAppWebView("curl/8.0"))
}
