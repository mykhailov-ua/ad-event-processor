package flow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContentTypeForPath_holdout(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "text/html; charset=utf-8", contentTypeForPath("index.html", ""))
	assert.Equal(t, "application/javascript; charset=utf-8", contentTypeForPath("", "app.js"))
	assert.Equal(t, "application/octet-stream", contentTypeForPath("data.bin", ""))
}
