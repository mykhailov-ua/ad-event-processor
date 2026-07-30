package ingestion

import (
	"context"
	"testing"

	"espx/internal/campaignmodel"
	"espx/internal/licensing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubLicenseRegistry struct {
	state licensing.LicenseState
}

func (s *stubLicenseRegistry) GetLicenseState() (licensing.LicenseState, licensing.Entitlements) {
	return s.state, licensing.Entitlements{}
}

func TestLicenseFilter_graceAllowsIngest(t *testing.T) {
	f := NewLicenseFilter(&stubLicenseRegistry{state: licensing.StateGrace})
	err := f.Check(context.Background(), &campaignmodel.Event{})
	assert.NoError(t, err)
}

func TestLicenseFilter_expiredRejects(t *testing.T) {
	f := NewLicenseFilter(&stubLicenseRegistry{state: licensing.StateExpired})
	err := f.Check(context.Background(), &campaignmodel.Event{})
	require.ErrorIs(t, err, ErrLicenseExpired)
}

func TestLicenseFilter_revokedRejects(t *testing.T) {
	f := NewLicenseFilter(&stubLicenseRegistry{state: licensing.StateRevoked})
	err := f.Check(context.Background(), &campaignmodel.Event{})
	require.ErrorIs(t, err, ErrLicenseExpired)
}

func TestLicenseFilter_offlineWarnAllowsIngest(t *testing.T) {
	f := NewLicenseFilter(&stubLicenseRegistry{state: licensing.StateOfflineWarn})
	err := f.Check(context.Background(), &campaignmodel.Event{})
	assert.NoError(t, err)
}

func TestLicenseFilter_offlineGraceAllowsIngest(t *testing.T) {
	f := NewLicenseFilter(&stubLicenseRegistry{state: licensing.StateOfflineGrace})
	err := f.Check(context.Background(), &campaignmodel.Event{})
	assert.NoError(t, err)
}

func TestFault_LicenseGraceIngestContinues(t *testing.T) {
	f := NewLicenseFilter(&stubLicenseRegistry{state: licensing.StateGrace})
	if err := f.Check(context.Background(), &campaignmodel.Event{}); err != nil {
		t.Fatalf("grace must allow ingest: %v", err)
	}
	t.Log("fault_proof fault=license_grace_ingest subsystem=ingestion state=GRACE")
}
