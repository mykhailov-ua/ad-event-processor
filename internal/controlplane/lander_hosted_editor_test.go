package controlplane

import (
	"testing"

	"ad-event-processor/internal/flow"

	"github.com/stretchr/testify/assert"
)

func TestHostedEditorFilePath_unescape(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "index.html", flow.HostedEditorFilePath("index.html"))
	assert.Equal(t, "assets/app.js", flow.HostedEditorFilePath("assets%2Fapp.js"))
	assert.Equal(t, "css/style.css", flow.HostedEditorFilePath("/css/style.css"))
}
