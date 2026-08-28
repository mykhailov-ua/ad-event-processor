package controlplane

import (
	"ad-event-processor/internal/flow"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisterHostedLanderRoutes_noServeMuxConflict(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	h := &flow.HTTPHandlers{Service: &Service{}}
	require.NotPanics(t, func() {
		h.RegisterHostedLanderRoutes(mux)
	})
}
