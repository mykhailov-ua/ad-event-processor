package trialregistry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRegistry_expireStale(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir+"/trial.json", 60*24*time.Hour)

	dep := "550e8400-e29b-41d4-a716-446655440000"
	past := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	future := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)

	require.NoError(t, reg.RecordPilotIssue(RecordInput{
		TelegramID:   "900",
		DeploymentID: dep,
		LicenseKey:   dep,
		ValidUntil:   past,
	}))
	require.NoError(t, reg.RecordPilotIssue(RecordInput{
		TelegramID:   "901",
		DeploymentID: "660e8400-e29b-41d4-a716-446655440001",
		LicenseKey:   "660e8400-e29b-41d4-a716-446655440001",
		ValidUntil:   future,
	}))

	n, err := reg.ExpireStale(now)
	require.NoError(t, err)
	require.Equal(t, 2, n)

	ok, err := reg.DeploymentHasStatus(dep, StatusExpired)
	require.NoError(t, err)
	require.True(t, ok)

	err = reg.CheckPilotEligible(CheckInput{TelegramID: "900"})
	require.ErrorIs(t, err, ErrTrialTelegramUsed)

	active, err := reg.DeploymentHasStatus("660e8400-e29b-41d4-a716-446655440001", StatusActive)
	require.NoError(t, err)
	require.True(t, active)

	err = reg.CheckPilotEligible(CheckInput{TelegramID: "902"})
	require.NoError(t, err)
}

func TestRegistry_markExpiredDeployment(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir+"/trial.json", 60*24*time.Hour)

	dep := "550e8400-e29b-41d4-a716-446655440000"
	future := time.Now().UTC().Add(7 * 24 * time.Hour)
	require.NoError(t, reg.RecordPilotIssue(RecordInput{
		TelegramID:   "800",
		DeploymentID: dep,
		LicenseKey:   dep,
		ValidUntil:   future,
	}))

	require.NoError(t, reg.MarkExpired(dep))

	err := reg.CheckPilotEligible(CheckInput{TelegramID: "800"})
	require.ErrorIs(t, err, ErrTrialTelegramUsed)
}

func TestRegistry_pilotToPaidConversionDeniesPilot(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir+"/trial.json", 60*24*time.Hour)

	dep := "550e8400-e29b-41d4-a716-446655440000"
	require.NoError(t, reg.RecordPilotIssue(RecordInput{
		TelegramID:   "700",
		DeploymentID: dep,
		LicenseKey:   dep,
		ValidUntil:   time.Now().UTC().Add(10 * 24 * time.Hour),
	}))
	require.NoError(t, reg.MarkConverted(dep))

	err := reg.CheckPilotEligible(CheckInput{TelegramID: "700", DeploymentID: "other-dep"})
	require.ErrorIs(t, err, ErrTrialTelegramUsed)
}
