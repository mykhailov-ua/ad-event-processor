package controlplane

import (
	"testing"

	ctrlhttp "ad-event-processor/internal/control/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminBootInjectRoundTrip(t *testing.T) {
	t.Parallel()
	raw := []byte(`<!doctype html><html><body><div id="root"></div></body></html>`)
	boot := AdminBootJSON{
		User: ctrlhttp.UserDTO{
			ID:          "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			Role:        ctrlhttp.RoleManager,
			CustomerID:  "00000000-0000-0000-0000-000000000001",
			Permissions: []string{"campaigns:read"},
		},
		Permissions: []string{"campaigns:read"},
	}
	out, err := injectAdminBoot(raw, boot)
	require.NoError(t, err)
	assert.Contains(t, string(out), "__BOOT__")
	assert.Contains(t, string(out), "6ba7b810-9dad-11d1-80b4-00c04fd430c8")
}
