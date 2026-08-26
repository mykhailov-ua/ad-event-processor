package controlplane

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisterHostedLanderRoutes_noServeMuxConflict(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	h := &FlowHTTPHandlers{Service: &Service{}}
	require.NotPanics(t, func() {
		h.RegisterHostedLanderRoutes(mux)
	})
}
