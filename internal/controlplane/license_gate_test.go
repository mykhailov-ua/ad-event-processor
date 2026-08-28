package controlplane

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLicenseFeatureAllowed_nilWatcherOpen(t *testing.T) {
	t.Parallel()
	activeLicenseWatcher.Store(nil)
	allowed, plan := licenseFeatureAllowed("openrtb")
	require.True(t, allowed)
	require.Empty(t, plan)
}

func TestWriteLicenseFeatureRequired_bodyShape(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	writeLicenseFeatureRequired(w, "openrtb", "pilot")
	require.Equal(t, http.StatusForbidden, w.Code)
	var body LicenseFeatureRequiredBody
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "feature_required", body.Error)
	require.Equal(t, "openrtb", body.FeatureKey)
	require.Equal(t, "pilot", body.PlanCode)
}
