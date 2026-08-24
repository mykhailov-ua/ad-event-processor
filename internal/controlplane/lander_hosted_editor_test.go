package controlplane

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHostedEditorFilePath_unescape(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "index.html", hostedEditorFilePath("index.html"))
	assert.Equal(t, "assets/app.js", hostedEditorFilePath("assets%2Fapp.js"))
	assert.Equal(t, "css/style.css", hostedEditorFilePath("/css/style.css"))
}
